package servertesting

type testAlphanumericGenerator struct {
	returnValues []string
	currentIdx   int
}

func NewTestAlphanumericGenerator(returnValues ...string) *testAlphanumericGenerator {
	return &testAlphanumericGenerator{returnValues: returnValues}
}

func (t *testAlphanumericGenerator) Generate(length int) string {
	if t.currentIdx >= len(t.returnValues) {
		t.currentIdx = 0
	}
	t.currentIdx++
	return t.returnValues[t.currentIdx-1]
}
