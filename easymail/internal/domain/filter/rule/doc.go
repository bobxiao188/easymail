// Package rule defines rule-engine domain entities, condition AST, and the extractor/content-plugin registry.
//
// Types here form the core of the antispam rule evaluation system:
//   - Rule, CustomFeature, BuiltinFeature  — rule/feature definitions
//   - CondNode, EvalConditionJSON            — condition AST parsing & evaluation
//   - FeatureExtractor, ContentPlugin         — port interfaces for extraction
//   - Register, ForStage, RunPlugins          — global registry
package rule

