/*
 * Copyright (c) 2026 easymail.my. All rights reserved.
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
 * GNU Affero General Public License for more details.
 *
 * For commercial licensing inquiries, please contact: 3680010825@qq.com
 *
 * Author: bob.xiao
 * License: AGPLv3
 */

package i18n

// Message identifiers for go-i18n bundles (active.*.json).
const (
	KeyAPISuccess = "api.success"

	KeyAuthHeaderInvalid      = "auth.header_invalid"
	KeyAuthTokenInvalid       = "auth.token_invalid"
	KeyAuthTokenInvalidSh     = "auth.token_invalid_short"
	KeyAuthUnauthorized       = "auth.unauthorized"
	KeyAuthInvalidCredentials = "auth.invalid_credentials"
	KeyAuthAccountInactive    = "auth.account_inactive"

	KeyWebmailAuthInvalidCredentials = "webmail.auth.invalid_credentials"
	KeyWebmailAuthLoginFailed        = "webmail.auth.login_failed"
	KeyWebmailAuthTokenError         = "webmail.auth.token_error"

	KeyWebmailContactsUnavailable      = "webmail.contacts.unavailable"
	KeyWebmailContactsInvalidGroupID   = "webmail.contacts.invalid_group_id"
	KeyWebmailContactsInvalidContactID = "webmail.contacts.invalid_contact_id"
	KeyWebmailContactsNotFound         = "webmail.contacts.not_found"
	KeyWebmailContactsDuplicate        = "webmail.contacts.duplicate"
	KeyWebmailContactsInvalidGroup     = "webmail.contacts.invalid_group"
	KeyWebmailContactsInvalidArgument  = "webmail.contacts.invalid_argument"

	KeyWebmailComposeInvalidMultipart     = "webmail.compose.invalid_multipart"
	KeyWebmailComposeAttachmentOpenFailed = "webmail.compose.attachment_open_failed"
	KeyWebmailComposeAttachmentReadFailed = "webmail.compose.attachment_read_failed"

	KeyErrBadRequest                          = "error.bad_request"
	KeyErrInvalidID                           = "error.invalid_id"
	KeyErrNotFoundAccount                     = "error.not_found.account"
	KeyErrNotFoundDomain                      = "error.not_found.domain"
	KeyErrNotFoundFeature                     = "error.not_found.feature"
	KeyErrNotFoundRule                        = "error.not_found.rule"
	KeyErrNotFoundRecord                      = "error.not_found.record"
	KeyErrDomainDeleted                       = "error.domain_deleted"
	KeyErrDomainAlreadyExists                 = "error.domain_already_exists"
	KeyErrDomainNotDeleted                    = "error.domain_not_deleted"
	KeyErrUsernameTaken                       = "error.username_taken"
	KeyErrAccountDisabled                     = "error.account_disabled"
	KeyErrPasswordMinLen                      = "error.password_min_length"
	KeyErrNotFoundUser                        = "error.not_found.user"
	KeyErrNotFoundClassifyModel               = "error.not_found.classify_model"
	KeyErrNotFoundModelSample                 = "error.not_found.model_sample"
	KeyErrNotFoundPublicSample                = "error.not_found.public_sample"
	KeyErrNotFoundPublicSampleCategory        = "error.not_found.public_sample_category"
	KeyErrModelSampleInvalid                  = "error.model_sample.invalid"
	KeyErrModelSampleBatchTooLarge            = "error.model_sample.batch_too_large"
	KeyErrClassifyModelSavePathRequired       = "error.classify_model.save_path_required"
	KeyErrClassifyModelNameInvalidFeatureKey  = "error.classify_model.name_invalid_feature_key"
	KeyErrClassifyModelNameConflictsBuiltin   = "error.classify_model.name_conflicts_builtin"
	KeyErrClassifyModelNameConflictsCustom    = "error.classify_model.name_conflicts_custom"
	KeyErrClassifyModelNameDuplicate          = "error.classify_model.name_duplicate"
	KeyErrClassifyModelRemoveFiles            = "error.classify_model.remove_files_failed"
	KeyErrClassifyModelActivationNotReady     = "error.classify_model.activation_not_ready"
	KeyErrFastTextExecutableNotConfigured     = "error.classify_model.fasttext_executable_not_configured"
	KeyErrClassifyModelModelRootNotConfigured = "error.classify_model.model_root_not_configured"
	KeyErrClassifyModelTrainNotFastText       = "error.classify_model.train_not_fasttext"
	KeyErrClassifyModelTrainAlreadyRunning    = "error.classify_model.train_already_running"
	KeyErrClassifyModelTrainNoSamples         = "error.classify_model.train_no_samples"
	KeyErrClassifyModelPredictTextRequired    = "error.classify_model.predict_text_required"
	KeyErrClassifyModelOnnxUploadNoFile       = "error.classify_model.onnx_upload_no_file"
	KeyErrClassifyModelOnnxInvalidExt         = "error.classify_model.onnx_invalid_extension"
	KeyErrClassifyModelOnnxSaveFailed         = "error.classify_model.onnx_save_failed"
	KeyErrClassifyModelDistilBERTUseMultipart = "error.classify_model.distilbert_use_multipart"
	KeyErrClassifyModelOnnxRequired           = "error.classify_model.onnx_required"
	KeyErrClassifyModelDirRenameFailed        = "error.classify_model.dir_rename_failed"
	KeyErrClassifyModelNotDistilBERT          = "error.classify_model.not_distilbert"
	KeyErrClassifyModelOnnxLabelsParse        = "error.classify_model.onnx_labels_parse"
	KeyErrClassifyModelMultipartParseFailed   = "error.classify_model.multipart_parse_failed"
	KeyErrClassifyModelMultipartNameTokenizer = "error.classify_model.multipart_name_tokenizer_required"
	KeyErrClassifyModelDistilBERTMaxTextLen   = "error.classify_model.distilbert_max_text_length_invalid"
	KeyErrClassifyModelMultipartLanguages     = "error.classify_model.multipart_languages_invalid"
	KeyErrClassifyModelMultipartEmailFields   = "error.classify_model.multipart_email_fields_invalid"
	KeyErrClassifyModelExportNoBin            = "error.classify_model.export_no_binary"
	KeyErrClassifyModelImportInvalidZip       = "error.classify_model.import_invalid_zip"
	KeyErrClassifyModelImportMissingConf      = "error.classify_model.import_missing_conf"
	KeyErrClassifyModelImportMissingBin       = "error.classify_model.import_missing_binary"
	KeyErrClassifyModelImportConfParse        = "error.classify_model.import_conf_parse_error"
	KeyErrClassifyModelImportNameConflict     = "error.classify_model.import_name_conflict"
	KeyErrClassifyModelImportWriteFailed      = "error.classify_model.import_write_failed"
	KeyErrClassifyModelImportAlgorithmMismatch = "error.classify_model.import_algorithm_mismatch"
	KeyFilterFeatureKeyClassifyModelReserved  = "filter.feature_key_classify_model_reserved"
	KeyErrProfileLoadFailed                   = "error.profile_load_failed"
	KeyErrInvalidLanguageParam                = "error.invalid_language_param"
	KeyErrInvalidSkinParam                    = "error.invalid_skin_param"
	KeyErrInternalServer                      = "error.internal_server"
	KeyErrOperationFailed                     = "error.operation_failed"
	KeyErrTrainingModelNameRequired           = "error.training.model_name_required"
	KeyErrTrainingMinClasses                  = "error.training.min_classes"
	KeyErrTrainingClassNameInvalid            = "error.training.class_name_invalid"
	KeyErrTrainingClassNameDuplicate          = "error.training.class_name_duplicate"
	KeyErrTrainingNoTags                      = "error.training.no_tags"
	KeyErrNewPasswordMustDiffer               = "error.new_password_must_differ"
	KeyErrInvalidCreateAccount                = "error.invalid_create_account"
	KeyErrInvalidDomainUsername               = "error.invalid_domain_username"
	KeyErrInvalidAccountPassword              = "error.invalid_account_password"
	KeyErrMailUserNotDeleted                   = "error.mail_user_not_deleted"
	KeyErrMailUserAlreadyExists                = "error.mail_user_already_exists"

	KeyFilterDBNotConfigured          = "filter.db_not_configured"
	KeyFilterFeatureKeyInvalid        = "filter.feature_key_invalid"
	KeyFilterTypeInvalid              = "filter.type_invalid"
	KeyFilterValueTypeInvalid         = "filter.value_type_invalid"
	KeyFilterConditionInvalid         = "filter.condition_json_invalid"
	KeyFilterActionInvalid            = "filter.action_invalid"
	KeyFilterCustomSpecInvalid        = "filter.custom_spec_invalid"
	KeyFilterClassifyModelFeatureDesc = "filter.classify_model_feature_desc"

	KeyDashboardErrServiceStatus = "dashboard.err.service_status"
	KeyDashboardErrMailDaily     = "dashboard.err.mail_daily"
	KeyDashboardErrMailMonthly   = "dashboard.err.mail_monthly"
	KeyDashboardErrTopDomain     = "dashboard.err.top_domain"
	KeyDashboardErrTopAddress    = "dashboard.err.top_address"
	KeyDashboardErrTopSenders    = "dashboard.err.top_senders"

	KeyDashboardSvcDovecot = "dashboard.svc.dovecot"
	KeyDashboardSvcMilter  = "dashboard.svc.milter"
	KeyDashboardSvcFilter  = "dashboard.svc.filter"
	KeyDashboardSvcLMTP    = "dashboard.svc.lmtp"
	KeyDashboardSvcAdmin   = "dashboard.svc.admin"
	KeyDashboardSvcWebmail = "dashboard.svc.webmail"
	KeyDashboardSvcIMAP    = "dashboard.svc.imap"

	KeyWebmailRecipientRequired   = "webmail.compose.recipient_required"
	KeyWebmailLocalOnly           = "webmail.compose.local_only"
	KeyWebmailInvalidEmail        = "webmail.compose.invalid_email"
	KeyWebmailBodyEmpty           = "webmail.compose.body_empty"
	KeyWebmailSMTPConnectFailed   = "webmail.compose.smtp_connect_failed"
	KeyWebmailSMTPSendFailed      = "webmail.compose.smtp_send_failed"
	KeyWebmailRecipientNotFound   = "webmail.compose.recipient_not_found"
	KeyWebmailSaveFailed          = "webmail.compose.save_failed"
	KeyWebmailWriteFileFailed     = "webmail.compose.write_file_failed"

	KeyLogHTTPListen   = "log.http_listen"
	KeyLogHTTPSListen  = "log.https_listen"
	KeyLogListen       = "log.listen"
	KeyLogListenLMTP   = "log.listen_lmtp"
	KeyLogListenMilter = "log.listen_milter"
	KeyLogListenIMAP   = "log.listen_imap"

	KeyLogLauncherStarted   = "log.launcher.started"
	KeyLogLauncherStopErr   = "log.launcher.stop_error"
	KeyLogLauncherStopped   = "log.launcher.stopped"
	KeyLogUsingConfig       = "log.using_config"
	KeyLogShutdownGraceful  = "log.shutdown_graceful"
	KeyLogServerError       = "log.server_error"
	KeyLogServerShutdownErr = "log.server_shutdown_error"

	KeyHealthDBNil      = "health.db_not_configured"
	KeyHealthRedisNil   = "health.redis_not_configured"
	KeyHealthMySQLError = "health.mysql_error"
	KeyHealthRedisError = "health.redis_error"

	KeyMailNotFound               = "mail.not_found"
	KeyMailForbidden              = "mail.forbidden"
	KeyMailInvalidArgument        = "mail.invalid_argument"
	KeyMailPurgeOrder             = "mail.purge_order"
	KeyMailFolderSystem           = "mail.folder_system"
	KeyMailFolderNotEmpty         = "mail.folder_not_empty"
	KeyMailInvalidFolderID        = "mail.invalid_folder_id"
	KeyMailInvalidMessageID       = "mail.invalid_message_id"
	KeyMailFolderNotFound         = "mail.folder_not_found"
	KeyMailInvalidAttachmentIndex = "mail.invalid_attachment_index"
	KeyMailNoChanges              = "mail.no_changes"
	KeyMailFolderIDRequired       = "mail.folder_id_required"
	KeyMailUnknownOp              = "mail.unknown_op"
	KeyMailInvalidDraftID         = "mail.invalid_draft_id"
	KeyMailInvalidRequestBody     = "mail.invalid_request_body"
	KeyMailInboxNotFound          = "mail.inbox_not_found"
	KeyMailKeywordRequired        = "mail.keyword_required"
	KeyMailInvalidFile            = "mail.invalid_file"
	KeyMailSenderEmailNotFound    = "mail.sender_email_not_found"
	KeyMailUserEmailNotFound      = "mail.user_email_not_found"

	KeyLabelInvalidLabelID  = "label.invalid_label_id"
	KeyLabelInvalidEmailID  = "label.invalid_email_id"
	KeyLabelInvalidRequest  = "label.invalid_request"

	KeyWebmailContactsDefaultGroupCannotModify = "webmail.contacts.default_group_cannot_modify"
	KeyWebmailContactsDefaultGroupCannotDelete = "webmail.contacts.default_group_cannot_delete"

	KeyWebmailAuthOldPasswordIncorrect = "webmail.auth.old_password_incorrect"

	// System folder display names (localized)
	KeyFolderNameInbox      = "folder.name.inbox"
	KeyFolderNameSent       = "folder.name.sent"
	KeyFolderNameDraft      = "folder.name.draft"
	KeyFolderNameTrash      = "folder.name.trash"
	KeyFolderNameSpam       = "folder.name.spam"
	KeyFolderNameQuarantine = "folder.name.quarantine"

	// Contact group display names (localized)
	KeyContactGroupDefault = "contact.group.default"
)
