package evalfixture

type Report struct {
	AccountID     string
	Region        string
	IncludeDrafts bool
	Compact       bool
	Notify        bool
	MaxRows       int
	Layout        string
}

func ExportReport(
	accountID string,
	region string,
	includeDrafts bool,
	compact bool,
	notify bool,
	options map[string]any,
) (Report, error) {
	maxRows, _ := options["max_rows"].(int)
	layout, _ := options["layout"].(string)
	return Report{
		AccountID:     accountID,
		Region:        region,
		IncludeDrafts: includeDrafts,
		Compact:       compact,
		Notify:        notify,
		MaxRows:       maxRows,
		Layout:        layout,
	}, nil
}
