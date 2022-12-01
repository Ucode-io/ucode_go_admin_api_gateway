package testing

import (
	"context"
	"fmt"
	"testing"
	"ucode/ucode_go_api_gateway/genproto/company_service"
)

func TestProjectCreate(t *testing.T) {
	project := company_service.CreateProjectRequest{
		Title:        "Object Builer II",
		K8SNamespace: "vault",
		CompanyId:    "7cf0cec4-0753-415c-a026-d658a7cd3fb6",
	}

	resp, err := projectClient.CreateProject(
		context.Background(),
		&company_service.CreateProjectRequest{
			Title:        project.Title,
			K8SNamespace: project.K8SNamespace,
			CompanyId:    project.CompanyId,
		},
	)

	if err != nil {
		fmt.Println(err)
	}

	fmt.Println("response :", resp)
}

func TestGetProjectById(t *testing.T) {
	companyID := "7cf0cec4-0753-415c-a026-d658a7cd3fb6"
	projectID := "255496e5-924e-48e5-bbfc-9228914bd407"

	resp, err := projectClient.GetProjectById(
		context.Background(),
		&company_service.GetProjectByIdRequest{
			ProjectId: projectID,
			CompanyId: companyID,
		},
	)

	if err != nil {
		t.Error("Error occured while getting company by ID : ", err.Error())
	}

	fmt.Println("Response : ", resp)
}

func TestGetProjectList(t *testing.T) {

	resp, err := projectClient.GetProjectList(
		context.Background(),
		&company_service.GetProjectListRequest{
			Limit:     2,
			Offset:    1,
			Search:    "",
			CompanyId: "",
		},
	)

	if err != nil {
		t.Error("Error occured while getting company list : ", err.Error())
	}

	fmt.Println("Response :", resp)

}

func TestUpdateProject(t *testing.T) {

	resp, err := projectClient.UpdateProject(
		context.Background(),
		&company_service.Project{
			CompanyId:    "7cf0cec4-0753-415c-a026-d658a7cd3fb6",
			ProjectId:    "d5ef7802-5efa-4f04-8c76-45b6d68a894d",
			K8SNamespace: "Albatta Warehouse Management System I Vaule",
			Title:        "Object Builder I",
		},
	)

	if err != nil {
		t.Error("Error occured while updating company :", err.Error())
	}

	fmt.Println("Response :", resp)
}

func TestDeleteProject(t *testing.T) {
	companyId := "7cf0cec4-0753-415c-a026-d658a7cd3fb6"
	projectID := "46f70f6d-950a-4f09-a256-c56cd733bc9f"
	_, err := projectClient.DeleteProject(
		context.Background(),
		&company_service.DeleteProjectRequest{
			CompanyId: companyId,
			ProjectId: projectID,
		},
	)

	if err != nil {
		t.Error("Error occured while deleting company :", err.Error())
	}

}
