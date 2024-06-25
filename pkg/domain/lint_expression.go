package domain

type LintExpression struct {
	Rule      string                       `json:"rule"`
	Type      string                       `json:"type"`
	Message   string                       `json:"message"`
	Path      string                       `json:"path"`
	Positions []LintExpressionItemPosition `json:"positions"`
}

type LintExpressionsByRule []LintExpression

func (a LintExpressionsByRule) Len() int           { return len(a) }
func (a LintExpressionsByRule) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a LintExpressionsByRule) Less(i, j int) bool { return a[i].Rule < a[j].Rule }

type OPALintExpression struct {
	Rule     string `json:"rule"`
	Type     string `json:"type"`
	Message  string `json:"message"`
	Path     string `json:"path"`
	DocIndex int    `json:"docIndex"`
	Field    string `json:"field"`
	Match    string `json:"match"`
}

type LintExpressionItemPosition struct {
	Start LintExpressionItemLinePosition `json:"start"`
}

type LintExpressionItemLinePosition struct {
	Line int `json:"line"`
}
