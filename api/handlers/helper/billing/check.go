package billing

import (
	"context"
	"errors"

	"ucode/ucode_go_api_gateway/config"
	pb "ucode/ucode_go_api_gateway/genproto/company_service"
	nb "ucode/ucode_go_api_gateway/genproto/new_object_builder_service"
	"ucode/ucode_go_api_gateway/services"
)

var ErrAssetLimitExceeded = errors.New("you have reached the asset size limit on your current plan. Please upgrade to upload more files")

func CheckAssetSizeLimit(ctx context.Context, companyServices services.CompanyServiceI, srvc services.ServiceManagerI, projectId, resourceEnvId string, newFileSizeBytes int64) error {
	type (
		projRes struct {
			fareId string
			err    error
		}
		usageRes struct {
			assetBytes int64
			err        error
		}
	)

	projCh := make(chan projRes, 1)
	usageCh := make(chan usageRes, 1)

	go func() {
		project, err := companyServices.Project().GetById(ctx, &pb.GetProjectByIdRequest{ProjectId: projectId})
		if err != nil {
			projCh <- projRes{err: err}
			return
		}
		projCh <- projRes{fareId: project.GetFareId()}
	}()

	go func() {
		usage, err := srvc.GoObjectBuilderService().ObjectBuilder().GetResourceUsage(ctx, &nb.GetResourceUsageRequest{ProjectId: resourceEnvId})
		if err != nil {
			usageCh <- usageRes{err: err}
			return
		}
		usageCh <- usageRes{assetBytes: usage.GetAssetSize()}
	}()

	pRes := <-projCh
	if pRes.err != nil {
		return pRes.err
	}

	if len(pRes.fareId) == 0 {
		return nil
	}

	uRes := <-usageCh
	if uRes.err != nil {
		return uRes.err
	}

	totalMB := int32((uRes.assetBytes + newFileSizeBytes) / 1024 / 1024)
	limitResp, err := companyServices.Billing().CompareFunction(ctx, &pb.CompareFunctionRequest{
		Type:   config.FARE_ASSET_SIZE,
		FareId: pRes.fareId,
		Count:  totalMB,
	})
	if err != nil {
		return err
	}

	if !limitResp.GetHasAccess() {
		return ErrAssetLimitExceeded
	}

	return nil
}
