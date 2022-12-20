package security

import (
	"errors"
	"strings"

	"github.com/dgrijalva/jwt-go"
)

type TokenInfo struct {
	ID               string
	Tables           []Table
	ProjectID        string
	ClientPlatformID string
	ClientTypeID     string
	RoleID           string
	// UserID           string
	// IP               string
	// Data             string
}

type Table struct {
	TableSlug string
	ObjectID  string
}

func ParseClaims(token string, secretKey string) (result TokenInfo, err error) {
	var ok bool
	var claims jwt.MapClaims

	claims, err = ExtractClaims(token, secretKey)
	if err != nil {
		return result, err
	}

	result.ID, ok = claims["id"].(string)
	if !ok {
		err = errors.New("cannot parse 'id' field")
		return result, err
	}

	result.RoleID, ok = claims["role_id"].(string)
	if !ok {
		err = errors.New("cannot parse 'role_id' field")
		return result, err
	}

	if claims["tables"] != nil {
		for _, item := range claims["tables"].([]interface{}) {
			var table Table
			table.ObjectID = item.(map[string]interface{})["object_id"].(string)
			table.TableSlug = item.(map[string]interface{})["table_slug"].(string)
			result.Tables = append(result.Tables, table)
		}
	}

	// projectID := claims["project_id"].(string)
	// clientPlatformID := claims["client_platform_id"].(string)
	// clientTypeID := claims["client_type_id"].(string)
	// userID := claims["user_id"].(string)
	// roleID := claims["role_id"].(string)
	// ip := claims["ip"].(string)
	// data := claims["data"].(string)

	return
}

// ExtractClaims extracts claims from given token
func ExtractClaims(tokenString string, tokenSecretKey string) (jwt.MapClaims, error) {
	var (
		token *jwt.Token
		err   error
	)

	token, err = jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// check token signing method etc
		return []byte(tokenSecretKey), nil
	})

	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !(ok && token.Valid) {
		return nil, errors.New("invalid token")
	}

	return claims, nil
}

// ExtractToken checks and returns token part of input string
func ExtractToken(bearer string) (token string, err error) {
	strArr := strings.Split(bearer, " ")
	if len(strArr) == 2 {
		return strArr[1], nil
	}
	return token, errors.New("wrong token format")
}
