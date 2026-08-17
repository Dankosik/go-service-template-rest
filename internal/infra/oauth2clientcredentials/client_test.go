package oauth2clientcredentials

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"
)

func TestTokenReuseRenewalAndRestart(t *testing.T) {
	t.Parallel()
	constructed, err := New(validTestConfig(), nil, slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if err := constructed.Close(t.Context()); err != nil {
		t.Fatalf("idle Client.Close() error = %v", err)
	}

	clock := newMovableClock(fixedProviderTime)
	provider := &scriptedAcquirer{steps: []acquisitionStep{
		{token: accessToken{value: "first", expiresAt: fixedProviderTime.Add(time.Minute)}},
		{token: accessToken{value: "second", expiresAt: fixedProviderTime.Add(100 * time.Second)}},
		{err: failure(FailureProviderUnavailable)},
		{token: accessToken{value: "after-restart", expiresAt: fixedProviderTime.Add(150 * time.Second)}},
	}}
	client := requireTestClient(t, validTestConfig(), testClientOptions{now: clock.Now, acquire: provider.acquire})

	first := requireOperationToken(t, client)
	if got, err := first.authorization(); err != nil || got != "first" {
		t.Fatalf("first authorization = %q, %v", got, err)
	}
	requireOperationToken(t, client)
	if got := provider.Calls(); got != 1 {
		t.Fatalf("provider calls for reusable token = %d, want 1", got)
	}

	clock.Advance(50 * time.Second)
	second := requireOperationToken(t, client)
	if got, err := second.authorization(); err != nil || got != "second" {
		t.Fatalf("renewed authorization = %q, %v", got, err)
	}
	if got := provider.Calls(); got != 2 {
		t.Fatalf("provider calls at expiry margin = %d, want 2", got)
	}

	clock.Advance(40 * time.Second)
	if _, err := client.resolve(t.Context()); err == nil {
		t.Fatal("failed replacement returned no error")
	} else {
		assertFailureClass(t, err, FailureProviderUnavailable)
	}
	if _, err := client.resolve(t.Context()); err == nil {
		t.Fatal("suppressed replacement failure returned no error")
	} else {
		assertFailureClass(t, err, FailureProviderUnavailable)
	}
	if got := provider.Calls(); got != 3 {
		t.Fatalf("provider calls inside failure window = %d, want 3", got)
	}

	restarted := requireTestClient(t, validTestConfig(), testClientOptions{now: clock.Now, acquire: provider.acquire})
	if got, err := requireOperationToken(t, restarted).authorization(); err != nil || got != "after-restart" {
		t.Fatalf("restart authorization = %q, %v", got, err)
	}
	if got := provider.Calls(); got != 4 {
		t.Fatalf("provider calls after restart = %d, want 4", got)
	}
}

func TestOperationTokenCannotRenewAcrossExpiryMargin(t *testing.T) {
	t.Parallel()
	clock := newMovableClock(fixedProviderTime)
	provider := &scriptedAcquirer{steps: []acquisitionStep{
		{token: accessToken{value: "fixed", expiresAt: fixedProviderTime.Add(30 * time.Second)}},
		{token: accessToken{value: "new-operation", expiresAt: fixedProviderTime.Add(time.Minute)}},
	}}
	client := requireTestClient(t, validTestConfig(), testClientOptions{now: clock.Now, acquire: provider.acquire})
	token := requireOperationToken(t, client)

	clock.Advance(20 * time.Second)
	if value, err := token.authorization(); value != "" || err == nil {
		t.Fatalf("authorization at margin = %q, %v; want token_unusable", value, err)
	} else {
		assertFailureClass(t, err, FailureTokenUnusable)
	}
	if got := provider.Calls(); got != 1 {
		t.Fatalf("provider calls for fixed operation = %d, want 1", got)
	}
	if value, err := requireOperationToken(t, client).authorization(); err != nil || value != "new-operation" {
		t.Fatalf("new operation authorization = %q, %v", value, err)
	}
	if got := provider.Calls(); got != 2 {
		t.Fatalf("provider calls for new operation = %d, want 2", got)
	}
}

func TestAcquisitionWavesPreserveCallerCancellation(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		clock := newMovableClock(fixedProviderTime)
		successEntered := make(chan struct{})
		successRelease := make(chan struct{})
		failureEntered := make(chan struct{})
		failureRelease := make(chan struct{})
		recoveryEntered := make(chan struct{})
		recoveryRelease := make(chan struct{})
		provider := &scriptedAcquirer{steps: []acquisitionStep{
			{
				token:   accessToken{value: "shared", expiresAt: fixedProviderTime.Add(time.Minute)},
				entered: successEntered,
				release: successRelease,
			},
			{err: failure(FailureProviderUnavailable), entered: failureEntered, release: failureRelease},
			{
				token:   accessToken{value: "recovered", expiresAt: fixedProviderTime.Add(2 * time.Minute)},
				entered: recoveryEntered,
				release: recoveryRelease,
			},
		}}
		client := requireTestClient(t, validTestConfig(), testClientOptions{now: clock.Now, acquire: provider.acquire})

		contexts := make([]context.Context, 6)
		cancels := make([]context.CancelFunc, 6)
		results := make(chan resolutionResult, len(contexts))
		for index := range contexts {
			contexts[index], cancels[index] = context.WithCancel(context.Background()) //nolint:fatcontext // Each caller needs an independent cancellation context.
		}
		t.Cleanup(func() {
			for _, cancel := range cancels {
				cancel()
			}
		})
		go resolveInto(contexts[0], client, results)
		<-successEntered
		for index := 1; index < len(contexts); index++ {
			go resolveInto(contexts[index], client, results)
		}
		synctest.Wait()
		if got := provider.Calls(); got != 1 {
			t.Fatalf("success-wave provider calls = %d, want 1", got)
		}
		cancels[0]()
		cancels[1]()
		synctest.Wait()
		for range 2 {
			result := <-results
			assertFailureClass(t, result.err, FailureCallerCanceled)
			if !errors.Is(result.err, context.Canceled) {
				t.Fatalf("canceled result = %v, want context.Canceled identity", result.err)
			}
		}
		close(successRelease)
		synctest.Wait()
		for range len(contexts) - 2 {
			result := <-results
			if result.err != nil || result.value != "shared" {
				t.Fatalf("live success-wave result = %#v", result)
			}
		}

		clock.Advance(50 * time.Second)
		failureResults := make(chan resolutionResult, 4)
		for range 4 {
			go resolveInto(context.Background(), client, failureResults)
		}
		<-failureEntered
		synctest.Wait()
		if got := provider.Calls(); got != 2 {
			t.Fatalf("failure-wave provider calls = %d, want 2", got)
		}
		close(failureRelease)
		synctest.Wait()
		for range 4 {
			assertFailureClass(t, (<-failureResults).err, FailureProviderUnavailable)
		}
		if _, err := client.resolve(context.Background()); err == nil {
			t.Fatal("post-failure caller returned no error")
		} else {
			assertFailureClass(t, err, FailureProviderUnavailable)
		}
		if got := provider.Calls(); got != 2 {
			t.Fatalf("provider calls inside failure window = %d, want 2", got)
		}

		clock.Advance(validTestConfig().AcquisitionTimeout)
		recoveryResults := make(chan resolutionResult, 4)
		for range 4 {
			go resolveInto(context.Background(), client, recoveryResults)
		}
		<-recoveryEntered
		synctest.Wait()
		if got := provider.Calls(); got != 3 {
			t.Fatalf("recovery-wave provider calls = %d, want 3", got)
		}
		close(recoveryRelease)
		synctest.Wait()
		for range 4 {
			result := <-recoveryResults
			if result.err != nil || result.value != "recovered" {
				t.Fatalf("recovery-wave result = %#v", result)
			}
		}
	})
}

func TestProviderWorkOutlivesCallersWithinItsBudget(t *testing.T) {
	t.Parallel()
	t.Run("caller cancellation", func(t *testing.T) {
		t.Parallel()
		synctest.Test(t, func(t *testing.T) {
			entered := make(chan struct{})
			release := make(chan struct{})
			providerCanceled := make(chan struct{})
			provider := &scriptedAcquirer{steps: []acquisitionStep{{
				token:    accessToken{value: "shared", expiresAt: time.Now().Add(time.Minute)},
				entered:  entered,
				release:  release,
				canceled: providerCanceled,
			}}}
			client := requireTestClient(t, validTestConfig(), testClientOptions{now: time.Now, acquire: provider.acquire})

			callerCtx, cancelCaller := context.WithCancel(context.Background())
			callerResult := make(chan resolutionResult, 1)
			go resolveInto(callerCtx, client, callerResult)
			<-entered
			cancelCaller()
			synctest.Wait()
			callerErr := (<-callerResult).err
			assertFailureClass(t, callerErr, FailureCallerCanceled)
			if !errors.Is(callerErr, context.Canceled) {
				t.Fatalf("caller error = %v, want context.Canceled identity", callerErr)
			}
			select {
			case <-providerCanceled:
				t.Fatal("caller cancellation reached provider context")
			default:
			}

			liveResult := make(chan resolutionResult, 1)
			go resolveInto(context.Background(), client, liveResult)
			synctest.Wait()
			close(release)
			synctest.Wait()
			if result := <-liveResult; result.err != nil || result.value != "shared" {
				t.Fatalf("live caller result = %#v", result)
			}
			if got := provider.Calls(); got != 1 {
				t.Fatalf("provider calls = %d, want 1", got)
			}
		})
	})

	t.Run("provider timeout", func(t *testing.T) {
		t.Parallel()
		synctest.Test(t, func(t *testing.T) {
			entered := make(chan struct{})
			release := make(chan struct{})
			providerCanceled := make(chan struct{})
			provider := &scriptedAcquirer{steps: []acquisitionStep{{
				entered: entered, release: release, canceled: providerCanceled,
			}}}
			cfg := validTestConfig()
			client := requireTestClient(t, cfg, testClientOptions{now: time.Now, acquire: provider.acquire})
			result := make(chan resolutionResult, 1)
			go resolveInto(context.Background(), client, result)
			<-entered
			time.Sleep(cfg.AcquisitionTimeout)
			synctest.Wait()
			<-providerCanceled
			assertFailureClass(t, (<-result).err, FailureProviderTimeout)
			if got := provider.Calls(); got != 1 {
				t.Fatalf("provider calls = %d, want 1", got)
			}
		})
	})
}

func TestClientCloseRetiresAndJoinsAcquisition(t *testing.T) {
	t.Parallel()
	t.Run("joins active wave and closes once", func(t *testing.T) {
		t.Parallel()
		synctest.Test(t, func(t *testing.T) {
			entered := make(chan struct{})
			releaseAfterCancel := make(chan struct{})
			providerCanceled := make(chan struct{})
			provider := &scriptedAcquirer{steps: []acquisitionStep{{
				entered: entered, release: make(chan struct{}), canceled: providerCanceled, waitAfterCancel: releaseAfterCancel,
			}}}
			var idleCloses atomic.Int64
			client := requireTestClient(t, validTestConfig(), testClientOptions{
				now: time.Now, acquire: provider.acquire, closeIdle: func() { idleCloses.Add(1) },
			})
			releaseProvider := sync.OnceFunc(func() { close(releaseAfterCancel) })
			t.Cleanup(releaseProvider)
			resolveResult := make(chan resolutionResult, 1)
			go resolveInto(context.Background(), client, resolveResult)
			<-entered

			closeResult := make(chan error, 1)
			go func() { closeResult <- client.Close(context.Background()) }()
			<-providerCanceled
			synctest.Wait()
			select {
			case err := <-closeResult:
				t.Fatalf("Close returned before provider joined: %v", err)
			default:
			}
			if _, err := client.resolve(context.Background()); err == nil {
				t.Fatal("retired client admitted a resolution")
			} else {
				assertFailureClass(t, err, FailureProviderUnavailable)
			}
			releaseProvider()
			synctest.Wait()
			if err := <-closeResult; err != nil {
				t.Fatalf("Close() error = %v", err)
			}
			assertFailureClass(t, (<-resolveResult).err, FailureProviderUnavailable)
			if err := client.Close(context.Background()); err != nil {
				t.Fatalf("repeated Close() error = %v", err)
			}
			expiredCtx, expire := context.WithCancel(context.Background())
			expire()
			if err := client.Close(expiredCtx); err != nil {
				t.Fatalf("completed Close() with expired context error = %v", err)
			}
			if got := idleCloses.Load(); got != 1 {
				t.Fatalf("idle-pool closes = %d, want 1", got)
			}
			client.mu.Lock()
			retained := client.token
			client.mu.Unlock()
			if retained != (accessToken{}) {
				t.Fatalf("retained token after Close = %#v", retained)
			}
		})
	})

	t.Run("shutdown context expires before join", func(t *testing.T) {
		t.Parallel()
		synctest.Test(t, func(t *testing.T) {
			entered := make(chan struct{})
			release := make(chan struct{})
			provider := &scriptedAcquirer{steps: []acquisitionStep{{
				entered: entered, release: release, ignoreCancellation: true,
			}}}
			var idleCloses atomic.Int64
			client := requireTestClient(t, validTestConfig(), testClientOptions{
				now: time.Now, acquire: provider.acquire, closeIdle: func() { idleCloses.Add(1) },
			})
			releaseProvider := sync.OnceFunc(func() { close(release) })
			t.Cleanup(releaseProvider)
			go func() { _, _ = client.resolve(context.Background()) }()
			<-entered

			shutdownCtx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			closeResult := make(chan error, 1)
			go func() { closeResult <- client.Close(shutdownCtx) }()
			time.Sleep(time.Second)
			synctest.Wait()
			assertFailureClass(t, <-closeResult, FailureProviderUnavailable)
			if got := idleCloses.Load(); got != 0 {
				t.Fatalf("idle-pool closes before join = %d, want 0", got)
			}

			releaseProvider()
			synctest.Wait()
			if err := client.Close(context.Background()); err != nil {
				t.Fatalf("Close() after provider join error = %v", err)
			}
			if got := idleCloses.Load(); got != 1 {
				t.Fatalf("idle-pool closes after join = %d, want 1", got)
			}
		})
	})
}

type resolutionResult struct {
	value string
	err   error
}

func resolveInto(ctx context.Context, client *Client, result chan<- resolutionResult) {
	token, err := client.resolve(ctx)
	if err != nil {
		result <- resolutionResult{err: err}
		return
	}
	value, err := token.authorization()
	result <- resolutionResult{value: value, err: err}
}

func requireOperationToken(t *testing.T, client *Client) operationToken {
	t.Helper()
	token, err := client.resolve(t.Context())
	if err != nil {
		t.Fatalf("resolve() error = %v", err)
	}
	return token
}
