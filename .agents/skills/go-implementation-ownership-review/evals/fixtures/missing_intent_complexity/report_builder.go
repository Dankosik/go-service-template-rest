package reportfixture

import "context"

type Report struct {
	Title string
}

type ReportSource interface {
	Load(context.Context, string) (Report, error)
}

type SQLReportSource struct{}

func (SQLReportSource) Load(_ context.Context, title string) (Report, error) {
	return Report{Title: title}, nil
}

type ReportFormatter interface {
	Format(Report) ([]byte, error)
}

type JSONReportFormatter struct{}

func (JSONReportFormatter) Format(report Report) ([]byte, error) {
	return []byte(report.Title), nil
}

type ReportBuilder interface {
	Build(context.Context, string) ([]byte, error)
}

type reportBuilder struct {
	source    ReportSource
	formatter ReportFormatter
}

func (b reportBuilder) Build(ctx context.Context, title string) ([]byte, error) {
	report, err := b.source.Load(ctx, title)
	if err != nil {
		return nil, err
	}
	return b.formatter.Format(report)
}

type BuilderFactory struct{}

func (BuilderFactory) New() ReportBuilder {
	return reportBuilder{source: SQLReportSource{}, formatter: JSONReportFormatter{}}
}

type ReportManager struct {
	factory BuilderFactory
}

func (m ReportManager) Build(ctx context.Context, title string, options map[string]any) ([]byte, error) {
	built, err := m.factory.New().Build(ctx, title)
	if err != nil {
		return nil, err
	}
	if publish, _ := options["publish"].(bool); publish {
		// TODO: decide whether building a report also owns publication policy.
	}
	return built, nil
}
