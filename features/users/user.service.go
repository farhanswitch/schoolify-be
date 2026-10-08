package users

import (
	"context"
	"fmt"
	"schoolify_be/configs"
	models "schoolify_be/models"
	utilities "schoolify_be/utilities"
	custom_errors "schoolify_be/utilities/errors"
	errUtility "schoolify_be/utilities/errors"
	"sort"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gopkg.in/square/go-jose.v2/jwt"
)

type userPostgresRepoInterface interface {
	GetUserByUsername(param models.UserLoginRequest) (models.User, errUtility.CustomError)
	RegisterUser(user models.UserRegisterRequest) (bool, errUtility.CustomError)
	UserLogin(user models.UserLoginRequest) (models.UserLoginData, errUtility.CustomError)
	GetAccessMenus(ctx context.Context, userID string) ([]models.UserAccessMenuResponse, errUtility.CustomError)
}

type redisUtilityInterface interface {
	SaveValue(ctx context.Context, key string, value interface{}, ttl time.Duration) error
	DeleteValue(ctx context.Context, key string) error
	GetValue(ctx context.Context, key string) (string, error)
}

type userService struct {
	repo  userPostgresRepoInterface
	redis redisUtilityInterface
}

var service userService

func (s userService) GetUserDataByUsernameSrvc(param models.UserLoginRequest) (models.User, errUtility.CustomError) {

	user, customErr := s.repo.GetUserByUsername(param)
	utilities.Log(utilities.INFO, "/api/auth/login", "GetUserDataByUsernameSrvc", param.Username, "Log Data", user)
	return user, customErr
}
func (s userService) RegisterUserSrvc(param models.UserRegisterRequest) (bool, errUtility.CustomError) {
	strID, err := uuid.NewV7()
	if err != nil {
		utilities.Log(utilities.ERROR, "/api/auth/register", "RegisterUserSrvc", param, "Error generating userID: "+err.Error(), err)
		return true, errUtility.CustomError{
			Code:          500,
			Message:       err.Error(),
			MessageToSend: "Internal Server Error",
		}
	}
	strRoleID, err := uuid.NewV7()
	if err != nil {
		utilities.Log(utilities.ERROR, "/api/auth/register", "RegisterUserSrvc", param, "Error generating userRoleID: "+err.Error(), err)
		return true, errUtility.CustomError{
			Code:          500,
			Message:       err.Error(),
			MessageToSend: "Internal Server Error",
		}
	}
	param.ID = strID
	param.RoleID = strRoleID

	if string(param.ConfirmPassword) != string(param.Password) {

		return true, errUtility.CustomError{
			Code:          400,
			Message:       "Password and Confirm Password must match!",
			MessageToSend: "Password and Confirm Password must match!",
		}
	}
	// Hash the password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(param.Password), bcrypt.DefaultCost)
	if err != nil {
		utilities.Log(utilities.ERROR, "/api/auth/register", "RegisterUserSrvc", param, "Error hashing password: "+err.Error(), err)
		return true, custom_errors.CustomError{
			Code:          500,
			Message:       err.Error(),
			MessageToSend: "Internal Server Error",
		}
	}
	param.Password = string(hashedPassword)
	return s.repo.RegisterUser(param)

}

func (s userService) UserLogin(param models.UserLoginRequest) (map[string]any, errUtility.CustomError) {
	loginData, errObj := s.repo.UserLogin(param)
	if errObj.Code > 0 {
		return map[string]any{}, errObj
	}
	err := bcrypt.CompareHashAndPassword([]byte(loginData.Password), []byte(param.Password))
	if err != nil {
		utilities.Log(utilities.ERROR, "/api/auth/login", "UserLoginSrvc", param, "Error bcrypt compare: "+err.Error(), err)
		return map[string]any{}, custom_errors.CustomError{
			Code:          500,
			Message:       err.Error(),
			MessageToSend: "Invalid username or password",
		}
	}
	claims := map[string]any{
		"username": loginData.Username,
		"id":       loginData.ID,
		"roleID":   loginData.RoleID,
		"roleName": loginData.RoleName,
		"Issuer":   "Schoolify",
		"Expiry":   jwt.NewNumericDate(time.Now().Add(time.Duration(configs.GetConfig().Service.SessionTime) * time.Second)),
		"IssuedAt": jwt.NewNumericDate(time.Now()),
	}
	strToken, err := utilities.JWEEncryptAES(claims, []byte(configs.GetConfig().Service.EncryptKey))
	if err != nil {
		return map[string]any{}, custom_errors.CustomError{
			Code:          500,
			Message:       err.Error(),
			MessageToSend: "Invalid username or password",
		}
	}
	redisUtil := s.redis
	if redisUtil == nil {
		redisUtil = utilities.RedisInstance()
	}

	err = redisUtil.DeleteValue(context.Background(), fmt.Sprintf("TOKEN_%s_1", loginData.Username))
	if err != nil {
		return map[string]any{}, custom_errors.CustomError{
			Code:          500,
			Message:       err.Error(),
			MessageToSend: "Invalid username or password"}
	}

	err = redisUtil.DeleteValue(context.Background(), fmt.Sprintf("TOKEN_%s_2", loginData.Username))
	if err != nil {
		return map[string]any{}, custom_errors.CustomError{
			Code:          500,
			Message:       err.Error(),
			MessageToSend: "Invalid username or password"}
	}
	err = redisUtil.SaveValue(context.Background(), fmt.Sprintf("TOKEN_%s_1", loginData.Username), strToken, time.Duration(configs.GetConfig().Service.SessionTime)*time.Second)
	if err != nil {
		return map[string]any{}, custom_errors.CustomError{
			Code:          500,
			Message:       err.Error(),
			MessageToSend: "Invalid username or password"}
	}
	claims["token"] = strToken
	claims["firstName"] = loginData.FirstName
	claims["lastName"] = loginData.LastName

	return claims, errObj

}

func (s userService) GetAccessMenusSrvc(ctx context.Context, userID string) ([]*models.UserAccessMenuResponse, errUtility.CustomError) {
	flatMenus, errObj := s.repo.GetAccessMenus(ctx, userID)
	if errObj.Code > 0 {
		return []*models.UserAccessMenuResponse{}, errObj
	}
	return buildUserAccessMenuTree(flatMenus), errUtility.CustomError{}
}

func buildUserAccessMenuTree(flatMenus []models.UserAccessMenuResponse) []*models.UserAccessMenuResponse {
	nodes := make(map[string]*models.UserAccessMenuResponse, len(flatMenus))
	for i := range flatMenus {
		node := flatMenus[i]
		node.Children = []*models.UserAccessMenuResponse{}
		if node.Endpoints == nil {
			node.Endpoints = []models.MenuApiResponse{}
		}
		nodes[node.ID] = &node
	}

	roots := make([]*models.UserAccessMenuResponse, 0)
	for _, node := range nodes {
		if node.ParentID != nil {
			if parent, found := nodes[*node.ParentID]; found {
				parent.Children = append(parent.Children, node)
				continue
			}
		}
		roots = append(roots, node)
	}
	sortUserAccessMenuTree(roots)
	appendLeafChildEndpoints(roots)
	return roots
}

func sortUserAccessMenuTree(nodes []*models.UserAccessMenuResponse) {
	sort.SliceStable(nodes, func(i, j int) bool {
		if nodes[i].SortOrder == nodes[j].SortOrder {
			return nodes[i].ID < nodes[j].ID
		}
		return nodes[i].SortOrder < nodes[j].SortOrder
	})
	for _, node := range nodes {
		sortUserAccessMenuTree(node.Children)
	}
}

// appendLeafChildEndpoints makes the final submenu level immediately usable:
// when every direct child is a leaf, the parent exposes all endpoints assigned
// to those children while the children remain in the response hierarchy.
func appendLeafChildEndpoints(nodes []*models.UserAccessMenuResponse) {
	for _, node := range nodes {
		appendLeafChildEndpoints(node.Children)
		if len(node.Children) == 0 {
			continue
		}

		allChildrenAreLeaves := true
		for _, child := range node.Children {
			if len(child.Children) > 0 {
				allChildrenAreLeaves = false
				break
			}
		}
		if !allChildrenAreLeaves {
			continue
		}

		seen := make(map[uuid.UUID]struct{}, len(node.Endpoints))
		for _, endpoint := range node.Endpoints {
			seen[endpoint.Id] = struct{}{}
		}
		for _, child := range node.Children {
			for _, endpoint := range child.Endpoints {
				if _, exists := seen[endpoint.Id]; exists {
					continue
				}
				seen[endpoint.Id] = struct{}{}
				node.Endpoints = append(node.Endpoints, endpoint)
			}
		}
	}
}

func (s userService) LogoutSrvc(ctx context.Context, username string) errUtility.CustomError {
	redisUtil := s.redis
	if redisUtil == nil {
		redisUtil = utilities.RedisInstance()
	}

	for _, key := range []string{fmt.Sprintf("TOKEN_%s_1", username), fmt.Sprintf("TOKEN_%s_2", username)} {
		if err := redisUtil.DeleteValue(ctx, key); err != nil {
			return errUtility.CustomError{
				Code:          500,
				Message:       err.Error(),
				MessageToSend: "Failed to logout",
			}
		}
	}
	return errUtility.CustomError{}
}
func factoryUserService(repo userPostgresRepoInterface) userService {
	if service.repo == nil {
		service = userService{repo: repo, redis: utilities.RedisInstance()}
	}
	return service
}
