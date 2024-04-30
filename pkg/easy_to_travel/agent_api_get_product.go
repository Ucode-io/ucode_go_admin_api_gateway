package easy_to_travel

import (
	"errors"
	"strings"
	"time"

	"ucode/ucode_go_api_gateway/config"
	"ucode/ucode_go_api_gateway/pkg/helper"
	"ucode/ucode_go_api_gateway/pkg/util"

	"github.com/spf13/cast"
)

func AgentApiGetProduct(data map[string]interface{}) (interface{}, error) {

	var (
		errorResponse struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		}

		response struct {
			Metadata map[string]interface{}   `json:"metadata"`
			Results  []map[string]interface{} `json:"results"`
		}

		metadata = cast.ToStringMap(data["metadata"])
		results  = cast.ToSlice(data["results"])

		offset = 0
		limit  = 20
		total  int

		startTime = cast.ToString(cast.ToStringMap(data["filters"])["startTime"])
		endTime   = cast.ToString(cast.ToStringMap(data["filters"])["endTime"])

		startTimeType time.Time
		endTimeType   time.Time
		startFull     time.Time
		endFull       time.Time
		expireDate    bool
		noFilterTime  bool
		err           error
	)

	if _, ok := metadata["offset"]; ok {
		offset = cast.ToInt(metadata["offset"])
	}

	if _, ok := metadata["limit"]; ok {
		limit = cast.ToInt(metadata["limit"])
	}

	if len(startTime) <= 0 {
		errorResponse.Code = 400
		errorResponse.Message = "Bad request."
		return errorResponse, err
	}

	if len(startTime) > 0 {
		if strings.Contains(startTime, "T") && strings.Contains(startTime, "Z") {
			startTimeType, err = time.Parse("2006-01-02T15:04:05Z", startTime)
			if err != nil {
				errorResponse.Code = 400
				errorResponse.Message = "Bad request."
				return errorResponse, err
			}

			if util.TruncateToStartOfDay(time.Now()).After(startTimeType) {
				expireDate = true
			}

			startFull = startTimeType
			startTime = startTimeType.Format("15:04")
			startTimeType, err = time.Parse(config.TIME_LAYOUT, startTime)
			if err != nil {
				errorResponse.Code = 400
				errorResponse.Message = "Bad request."
				return errorResponse, err
			}
		} else {
			startTimeType, err = time.Parse("2006-01-02", startTime)
			if err != nil {
				errorResponse.Code = 400
				errorResponse.Message = "Bad request."
				return errorResponse, err
			}

			if util.TruncateToStartOfDay(time.Now()).After(startTimeType) {
				expireDate = true
			}

			startFull = startTimeType
			startTime = "00:00"
			startTimeType, err = time.Parse(config.TIME_LAYOUT, startTime)
			if err != nil {
				errorResponse.Code = 400
				errorResponse.Message = "Bad request."
				return errorResponse, err
			}
		}
	}

	if len(endTime) > 0 {
		if strings.Contains(endTime, "T") && strings.Contains(endTime, "Z") {
			endTimeType, err = time.Parse("2006-01-02T15:04:05Z", endTime)
			if err != nil {
				errorResponse.Code = 400
				errorResponse.Message = "Bad request."
				return errorResponse, err
			}

			// endTimeType = endTimeType.Add(time.Hour * time.Duration(airportUTC))

			endFull = endTimeType
			endTime = endTimeType.Format("15:04")
			endTimeType, err = time.Parse(config.TIME_LAYOUT, endTime)
			if err != nil {
				errorResponse.Code = 400
				errorResponse.Message = "Bad request."
				return errorResponse, err
			}
		} else {
			endTimeType, err = time.Parse("2006-01-02", endTime)
			if err != nil {
				errorResponse.Code = 400
				errorResponse.Message = "Bad request."
				return errorResponse, err
			}

			endFull = endTimeType
			endTime = "23:59"
			endTimeType, err = time.Parse(config.TIME_LAYOUT, endTime)
			if err != nil {
				errorResponse.Code = 400
				errorResponse.Message = "Bad request."
				return errorResponse, err
			}
		}
	} else {
		endFull = startFull
		endTime = "23:59"
		endTimeType, err = time.Parse(config.TIME_LAYOUT, endTime)
		if err != nil {
			errorResponse.Code = 400
			errorResponse.Message = "Bad request."
			return errorResponse, err
		}
	}

	if startTime == "00:00" && endTime == "23:59" {
		noFilterTime = true
	}

	if startFull.After(endFull) {
		response.Results = []map[string]interface{}{}
		response.Metadata = map[string]interface{}{"count": 0, "limit": limit, "offset": offset, "total": 0}
		return response, nil
	}

	var filterProduct = []map[string]interface{}{}
	for ind := range results {
		var timeData = map[string]interface{}{
			"startTimeType": startTimeType,
			"endTimeType":   endTimeType,
			"startFull":     startFull,
			"endFull":       endFull,
			"noFilterTime":  noFilterTime,
			"expireDate":    expireDate,
		}

		var product = cast.ToStringMap(results[ind])

		noTimeRange, err := EasyToTravelAgentApiGetProductAttributes(product, timeData)
		if err != nil {
			errorResponse.Code = 400
			errorResponse.Message = "Bad request."
			return errorResponse, err
		}

		if noTimeRange {
			continue
		}

		filterProduct = append(filterProduct, product)
	}

	total = len(filterProduct)

	if limit+offset > 0 {
		if offset <= len(filterProduct) {
			filterProduct = filterProduct[offset:]
		} else {
			errorResponse.Code = 404
			errorResponse.Message = "Wrong pagination parameters."
			return errorResponse, errors.New("Wrong pagination parameters.")
		}

		if limit > 0 {
			if limit <= len(filterProduct) {
				filterProduct = filterProduct[:limit]
			}
		}
	}

	response.Results = filterProduct
	response.Metadata = map[string]interface{}{"count": len(filterProduct), "limit": limit, "offset": offset, "total": total}
	return response, nil
}

func EasyToTravelAgentApiGetProductAttributes(product map[string]interface{}, timeData map[string]interface{}) (bool, error) {

	if cast.ToBool(timeData["expireDate"]) {
		return true, nil
	}

	// Get service time
	{
		var (
			serviceTimes     = cast.ToSlice(product["service_times"])
			serviceTimesData = []map[string]interface{}{}
			existTimeRange   bool

			startFull    = cast.ToTime(timeData["startFull"])
			endFull      = cast.ToTime(timeData["endFull"])
			startTime    = cast.ToTime(timeData["startTimeType"])
			endTime      = cast.ToTime(timeData["endTimeType"])
			noFilterTime = cast.ToBool(timeData["noFilterTime"])

			weekdays    = helper.GetWeekdayRange(startFull, endFull)
			setWeekDays = helper.RemoveDuplicateStrings(weekdays)
		)

		for i := 0; i < len(serviceTimes); i++ {
			serviceTimeObj := cast.ToStringMap(serviceTimes[i])

			if !helper.Contains(setWeekDays, strings.ToLower(cast.ToStringSlice(serviceTimeObj["day"])[0])) {
				continue
			}

			serviceTimesData = append(serviceTimesData, map[string]interface{}{
				"day":   cast.ToStringSlice(serviceTimeObj["day"])[0],
				"open":  cast.ToString(serviceTimeObj["opening_hour"]),
				"close": cast.ToString(serviceTimeObj["closing_hour"]),
			})

			openTime, err := time.Parse(config.TIME_LAYOUT, cast.ToString(serviceTimeObj["opening_hour"]))
			if err != nil {
				return false, err
			}
			openTime = openTime.Add(-1 * time.Minute)

			closeTime, err := time.Parse(config.TIME_LAYOUT, cast.ToString(serviceTimeObj["closing_hour"]))
			if err != nil {
				return false, err
			}
			closeTime = closeTime.Add(1 * time.Minute)

			if openTime.Before(startTime) && closeTime.After(endTime) {
				existTimeRange = true
			}
		}

		if !existTimeRange && !noFilterTime {
			return true, nil
		}

		if len(serviceTimesData) <= 0 {
			return true, nil
		}

		product["workingTime"] = serviceTimesData
	}

	helper.DeleteKeys(product, "guid", "service_times")

	return false, nil
}
