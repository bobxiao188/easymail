package classifier

import (
	"strings"
)

type PredictorInput struct {
	Text          string
	LanguageCodes []string
}

func BuildPredictorInput(raw string, langs []string) PredictorInput {
	return PredictorInput{
		Text:          strings.TrimSpace(raw),
		LanguageCodes: append([]string(nil), langs...),
	}
}

type LabelScore struct {
	Label       string  `json:"label"`
	Probability float64 `json:"probability"`
}

type Prediction struct {
	ModelID        string       `json:"modelId"`
	ModelName      string       `json:"modelName"`
	TopLabel       string       `json:"topLabel"`
	TopProbability float64      `json:"topProbability"`
	Distribution   []LabelScore `json:"distribution"`
	Err            string       `json:"predictError,omitempty"`
}

// ModelRuntime is a runtime projection of a Model row for opening predictors.
type ModelRuntime struct {
	ID            string
	Name          string
	Algorithm     Algorithm
	SavePath      string
	MaxTextLength int
	Params        ModelParams
}
