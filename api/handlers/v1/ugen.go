package v1

import (
	"sync"

	"ucode/ucode_go_api_gateway/api/models"
	"ucode/ucode_go_api_gateway/api/status_http"
	auth "ucode/ucode_go_api_gateway/genproto/auth_service"
	pb "ucode/ucode_go_api_gateway/genproto/company_service"

	"github.com/gin-gonic/gin"
)

// GetUgenUserProjects godoc
// @Security ApiKeyAuth
// @ID get_ugen_user_projects
// @Router /v1/ugen/user-projects [GET]
// @Summary Get Ugen User Projects
// @Description Returns companies and their projects the user has access to.
// @Description If a company has an is_ugen=true project, only that project is returned (personal fork).
// @Tags Ugen
// @Produce json
// @Success 200 {object} status_http.Response{data=models.UgenUserProjectsResp}
// @Failure 401 {object} status_http.Response{data=string}
// @Failure 500 {object} status_http.Response{data=string}
func (h *HandlerV1) GetUgenUserProjects(c *gin.Context) {
	var ctx = c.Request.Context()

	authInfo, err := h.adminAuthInfo(c)
	if err != nil {
		return
	}
	userId := authInfo.GetUserIdAuth()
	if userId == "" {
		h.HandleResponse(c, status_http.Unauthorized, "unauthorized")
		return
	}

	userProjects, err := h.authService.User().GetUserProjects(ctx, &auth.UserPrimaryKey{Id: userId})
	if err != nil {
		h.HandleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	var (
		companies = userProjects.GetCompanies()
		result    = models.UgenUserProjectsResp{
			Companies: make([]models.UgenCompanyResp, len(companies)),
		}

		wg       sync.WaitGroup
		mu       sync.Mutex
		firstErr error
	)

	for i, uc := range companies {
		wg.Add(1)
		go func(idx int, uc *auth.UserCompany) {
			defer wg.Done()

			companyResp, err := h.companyServices.Company().GetById(ctx, &pb.GetCompanyByIdRequest{Id: uc.GetId()})
			if err != nil {
				mu.Lock()
				if firstErr == nil {
					firstErr = err
				}
				mu.Unlock()
				return
			}

			var (
				company = companyResp.GetCompany()

				projectIds = uc.GetProjectIds()
				projects   = make([]models.UgenProjectData, len(projectIds))

				projWg sync.WaitGroup
			)

			for j, projId := range projectIds {
				projWg.Add(1)
				go func(jdx int, projId string) {
					defer projWg.Done()

					info, err := h.companyServices.Project().GetById(
						ctx, &pb.GetProjectByIdRequest{
							ProjectId: projId,
							CompanyId: uc.GetId(),
						},
					)
					if err != nil {
						return
					}

					ugenStatus, _ := h.companyServices.Project().GetProjectUgenStatus(
						ctx, &pb.GetProjectUgenStatusRequest{
							ProjectId: projId,
							CompanyId: uc.GetId(),
						},
					)

					envList, _ := h.companyServices.Environment().GetList(
						ctx, &pb.GetEnvironmentListRequest{
							ProjectId: projId,
							Search:    "Production",
							Limit:     1,
						},
					)

					var envId = envList.GetEnvironments()[0].GetId()

					projects[jdx] = models.UgenProjectData{
						Id:            info.GetProjectId(),
						Title:         info.GetTitle(),
						Logo:          info.GetLogo(),
						IsUgen:        ugenStatus.GetIsUgen(),
						EnvironmentId: envId,
					}
				}(j, projId)
			}
			projWg.Wait()

			ugenIdx := -1

			for k, p := range projects {
				if p.IsUgen {
					ugenIdx = k
					break
				}
			}

			companyOut := models.UgenCompanyResp{
				Id:              company.GetId(),
				Name:            company.GetName(),
				Logo:            company.GetLogo(),
				HasPersonalFork: ugenIdx >= 0,
				Projects:        make([]models.UgenProjectResp, 0, len(projects)),
			}

			if ugenIdx >= 0 {
				p := projects[ugenIdx]
				companyOut.Projects = append(companyOut.Projects, models.UgenProjectResp{
					Id:            p.Id,
					Title:         p.Title,
					Logo:          p.Logo,
					IsUgen:        true,
					EnvironmentId: p.EnvironmentId,
				})
			} else {
				for _, p := range projects {
					if p.Id == "" {
						continue
					}
					companyOut.Projects = append(companyOut.Projects, models.UgenProjectResp{
						Id:            p.Id,
						Title:         p.Title,
						Logo:          p.Logo,
						IsUgen:        false,
						EnvironmentId: p.EnvironmentId,
					})
				}
			}

			result.Companies[idx] = companyOut
		}(i, uc)
	}

	wg.Wait()

	if firstErr != nil {
		h.HandleResponse(c, status_http.GRPCError, firstErr.Error())
		return
	}

	h.HandleResponse(c, status_http.OK, result)
}
