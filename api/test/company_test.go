package testing

import (
	"context"
	"testing"
	"ucode/ucode_go_api_gateway/api/models"
	"ucode/ucode_go_api_gateway/genproto/company_service"
)

func TestCompanyCreate(t *testing.T) {
	company := models.CompanyCreateRequest{
		Name:        "Parfume Gallery",
		Logo:        "Logo",
		Description: "Description",
	}

	_, _ = companyClient.Create(
		context.Background(),
		&company_service.CreateCompanyRequest{
			Title:       company.Name,
			Logo:        company.Logo,
			Description: company.Description,
		},
	)

}

func TestCompanyGetByID(t *testing.T) {
	companyID := "51753b96-2f2d-4bdd-aba0-2c3d005aebf6"

	_, err := companyClient.GetById(
		context.Background(),
		&company_service.GetCompanyByIdRequest{
			Id: companyID,
		},
	)

	if err != nil {
		t.Error("Error occured while getting company by ID : ", err.Error())
	}

}

func TestCompanyList(t *testing.T) {

	_, err := companyClient.GetList(
		context.Background(),
		&company_service.GetCompanyListRequest{
			Limit:  2,
			Offset: 1,
			Search: "",
		},
	)

	if err != nil {
		t.Error("Error occured while getting company list : ", err.Error())
	}

}

func TestUpdateCompany(t *testing.T) {

	_, err := companyClient.Update(
		context.Background(),
		&company_service.Company{
			Id:          "7cf0cec4-0753-415c-a026-d658a7cd3fb6",
			Name:        "Albatta Warehouse Management System",
			Logo:        "https://www.company.com/logo.png",
			Description: "This is the company description",
		},
	)

	if err != nil {
		t.Error("Error occured while updating company :", err.Error())
	}

}

func TestDeleteCompany(t *testing.T) {
	companyId := "7cf0cec4-0753-415c-a026-d658a7cd3fb6"
	_, err := companyClient.Delete(
		context.Background(),
		&company_service.DeleteCompanyRequest{
			Id: companyId,
		},
	)

	if err != nil {
		t.Error("Error occured while deleting company :", err.Error())
	}

}
