package helper

import (
	"strings"
	"ucode/ucode_go_api_gateway/genproto/company_service"
	"ucode/ucode_go_api_gateway/services"
)

type MatchingData struct {
	ProjectId string
	EnvId     string
	Path      string
}

func FindUrlTo(res *company_service.GetListRedirectUrlRes, data MatchingData, comp services.CompanyServiceI) (string, error) {

	// start := time.Now()
	//fmt.Println("RES::::::::::::::::::::::", res)

	for _, v := range res.GetRedirectUrls() {
		m := make(map[string]string)
		from := strings.Split(v.From, "/")
		to := v.To
		path := strings.Split(data.Path, "/")
		isEqual := true

		if len(path) != len(from) {
			continue
		}

		for i, el := range from {
			if len(el) >= 1 && el[0] == '{' && el[len(el)-1] == '}' {
				m[el] = path[i]
			} else {
				if el != path[i] {
					isEqual = false
					break
				}
			}
		}

		if isEqual {
			for i, el := range m {
				to = strings.Replace(to, i, el, 1)
			}
			// fmt.Println("to::::::::::::::::::", to)
			return to, nil
		}
	}

	// fmt.Println("time in FindUrlTo::::::", time.Since(start).Milliseconds())
	return data.Path, nil
}

// something/{id}/{id} regex ^something/([^/]+)$
// get-list/{id}
// something/abcdfg/abcdfg
