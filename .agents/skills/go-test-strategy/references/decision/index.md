# Reference Selector

References sharpen a triggered proof decision after accepted behavior is reconstructed; they neither define scope nor act as checklists.

| Symptom | Load | Behavior change |
| --- | --- | --- |
| The test being written has an unclear proving layer or failure observable, or broader coverage is proposed for safety. | [proof-obligations.md](proof-obligations.md) | Makes the model choose the smallest boundary that can observe the failure and give the test a discriminating assertion, instead of escalating the level because the requirement matters and writing rows that only expect an error. |
| The proof target is a client-visible, durable, cached, failing, or cross-service boundary. | [boundary-observables.md](boundary-observables.md) | Makes the model name what a mock or a success response cannot fake — durable rows, origin calls, exactly one side effect, lifecycle markers — instead of treating a 200 or a recorded mock call as boundary proof. |
| Validation must be named, or the reported evidence may not exercise the changed surface. | [validation-commands.md](validation-commands.md) | Makes the model pick the command that would fail for this regression instead of offering `go test ./...` or demanding the full CI aggregate. |
