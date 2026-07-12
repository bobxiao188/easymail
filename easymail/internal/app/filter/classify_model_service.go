package filter

import (
	"context"
	"mime/multipart"

	"easymail/internal/domain/filter/classifier"
)

// ClassifyModelReader provides read-only access to classify model definitions.
type ClassifyModelReader interface {
	List(ctx context.Context, keyword, algorithm string, status *int, page, pageSize int) ([]classifier.Model, int64, error)
	GetByID(ctx context.Context, id int64) (*classifier.Model, error)
}

// ClassifyModelWriter provides CRUD operations for classify model definitions.
type ClassifyModelWriter interface {
	Create(ctx context.Context, name, algorithm, tokenizer string, languages classifier.Languages, savePath string, maxTextLength int, emailFields classifier.EmailFields, params string) error
	CreateDistilBERTWithONNXFile(ctx context.Context, name, tokenizer string, languages classifier.Languages, maxTextLength int, emailFields classifier.EmailFields, params string, onnx *multipart.FileHeader) error
	Update(ctx context.Context, id int64, name, algorithm, tokenizer string, languages classifier.Languages, savePath string, maxTextLength int, emailFields classifier.EmailFields, enabled bool, params string) error
	UpdateDistilBERTWithOptionalONNXFile(ctx context.Context, id int64, name, tokenizer string, languages classifier.Languages, maxTextLength int, emailFields classifier.EmailFields, enabled bool, params string, onnx *multipart.FileHeader) error
	Delete(ctx context.Context, id int64) error
}

// ClassifyModelTrainer provides model training operations.
type ClassifyModelTrainer interface {
	StartFastTextTraining(ctx context.Context, id int64) error
	Predict(ctx context.Context, id int64, text string, languageCodes []string) (classifier.Prediction, error)
}

// ClassifyModelSampleManager provides sample management operations for a model.
type ClassifyModelSampleManager interface {
	ListModelSamples(ctx context.Context, classifyModelID int64, keyword, labelFilter string, page, pageSize int) ([]classifier.Sample, int64, error)
	ListModelSampleLabels(ctx context.Context, classifyModelID int64) ([]string, error)
	ExportModelSamplesTrainTxt(ctx context.Context, classifyModelID int64) ([]byte, error)
	CreateModelSamples(ctx context.Context, classifyModelID int64, items []ModelSampleInput) error
	UpdateModelSample(ctx context.Context, classifyModelID, sampleID int64, text, label string) error
	DeleteModelSample(ctx context.Context, classifyModelID, sampleID int64) error
}

// ModelSampleInput is one row for CreateModelSamples.
type ModelSampleInput struct {
	Text  string
	Label string
}

// ClassifyModelExporter provides model export/import operations.
type ClassifyModelExporter interface {
	ExportModel(ctx context.Context, id int64) ([]byte, string, error)
	ImportModel(ctx context.Context, zipFile *multipart.FileHeader, expectedAlgorithm string) error
}

// ClassifyModelService is the full interface combining all sub-interfaces.
type ClassifyModelService interface {
	ClassifyModelReader
	ClassifyModelWriter
	ClassifyModelTrainer
	ClassifyModelSampleManager
	ClassifyModelExporter
}
