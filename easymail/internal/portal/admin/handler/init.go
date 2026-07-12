package handler

import (
	appAdmin "easymail/internal/app/admin"
	filterService "easymail/internal/app/filter"
	managementSvc "easymail/internal/app/management"
	"easymail/pkg/logger/easylog"
)

type Handler struct {
	storageIDs                    []int
	authenticationService         appAdmin.AuthenticationService
	mailAccountService            managementSvc.MailUserService
	mailDomainService             managementSvc.MailDomainService
	provisionService              managementSvc.UserProvisionService
	spamModelService              filterService.ClassifyModelService
	publicSampleService           filterService.PublicSampleService
	publicSampleCategoryService   filterService.PublicSampleCategoryService
	scannerService                appAdmin.FilterAdminService
	dashboardService              appAdmin.DashboardService
	postfixService                managementSvc.PostfixConfigService
	trainingService               filterService.TrainingService
	log                           *easylog.Logger
}

func New(
	storageIDs []int,
	authSvc appAdmin.AuthenticationService,
	mailDomainSvc managementSvc.MailDomainService,
	mailAccountSvc managementSvc.MailUserService,
	provisionSvc managementSvc.UserProvisionService,
	spamModelSvc filterService.ClassifyModelService,
	publicSampleSvc filterService.PublicSampleService,
	publicSampleCategorySvc filterService.PublicSampleCategoryService,
	scannerSvc appAdmin.FilterAdminService,
	dashboardSvc appAdmin.DashboardService,
	postfixSvc managementSvc.PostfixConfigService,
	trainingSvc filterService.TrainingService,
	logger *easylog.Logger,
) *Handler {
	return &Handler{
		storageIDs:                    storageIDs,
		authenticationService:         authSvc,
		mailDomainService:             mailDomainSvc,
		mailAccountService:            mailAccountSvc,
		provisionService:              provisionSvc,
		spamModelService:              spamModelSvc,
		publicSampleService:           publicSampleSvc,
		publicSampleCategoryService:   publicSampleCategorySvc,
		scannerService:                scannerSvc,
		dashboardService:              dashboardSvc,
		postfixService:                postfixSvc,
		trainingService:               trainingSvc,
		log:                           logger,
	}
}