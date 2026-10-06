package apis

import (
	"context"
	"net/http"
	"strconv"

	models "schoolify_be/models"
	repoPostgres "schoolify_be/repositories/postgresql"
	utilities "schoolify_be/utilities"
	errUtility "schoolify_be/utilities/errors"

	"github.com/go-playground/validator/v10"
)

var validate *validator.Validate

func init() {
	validate = validator.New()
}

type apiServiceInterface interface {
	GetAllSrvc(ctx context.Context, param models.GetAllApiRequest) ([]models.ApiResponse, int, errUtility.CustomError)
}

type apiController struct {
	service apiServiceInterface
}

func (c apiController) getAllCtrl(w http.ResponseWriter, r *http.Request) {
	reqID := utilities.GetReqID(r.Context())
	path := r.URL.Path

	pageStr := r.URL.Query().Get("paginationPage")
	rowsStr := r.URL.Query().Get("paginationRows")
	sortField := r.URL.Query().Get("sortField")
	sortOrder := r.URL.Query().Get("sortOrder")
	search := r.URL.Query().Get("search")

	page := 1
	if pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil {
			page = p
		}
	}

	rows := 10
	if rowsStr != "" {
		if rs, err := strconv.Atoi(rowsStr); err == nil {
			rows = rs
		}
	}

	if sortField == "" {
		sortField = "created_at"
	}
	if sortOrder == "" {
		sortOrder = "DESC"
	}

	req := models.GetAllApiRequest{
		Page:      page,
		Rows:      rows,
		SortField: sortField,
		SortOrder: sortOrder,
		Search:    search,
	}

	// Logging request payload
	utilities.Log(reqID, utilities.DEBUG, path, "apiController.getAllCtrl", req, "incoming request", nil)

	if err := validate.Struct(req); err != nil {
		errMap := errUtility.ParseError(err)
		utilities.Log(reqID, utilities.WARN, path, "apiController.getAllCtrl", req, "validation failed", errMap)
		utilities.WriteJSON(w, http.StatusBadRequest, map[string]interface{}{
			"message": "Validation failed",
			"errors":  errMap,
		})
		return
	}

	list, count, errObj := c.service.GetAllSrvc(r.Context(), req)
	if errObj.Code > 0 {
		errObj.Compile()
		statusCode := int(errObj.Code)

		logLevel := utilities.ERROR
		if statusCode < 500 {
			logLevel = utilities.WARN
		}
		utilities.Log(reqID, logLevel, path, "apiController.getAllCtrl", req, "service execution failed", errObj)

		if statusCode == 500 {
			utilities.WriteJSON(w, statusCode, map[string]interface{}{
				"message": "Failed to get list api",
				"errors":  "Internal Server Error",
			})
			return
		} else {
			errKey := errObj.Property
			if errKey == "" {
				errKey = "error"
			}
			utilities.WriteJSON(w, statusCode, map[string]interface{}{
				"message": "Failed to get list api",
				"errors": map[string]string{
					errKey: errObj.MessageToSend,
				},
			})
			return
		}
	}

	utilities.Log(reqID, utilities.DEBUG, path, "apiController.getAllCtrl", nil, "success", map[string]interface{}{"count": count})
	if list == nil {
		list = []models.ApiResponse{}
	}
	utilities.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"data": map[string]interface{}{
			"listData": list,
			"count":    count,
		},
		"message": "Success get list api",
	})
}

func (c apiController) GetAllCtrl(w http.ResponseWriter, r *http.Request) {
	c.getAllCtrl(w, r)
}

var controller apiController

func factoryApiController(repo repoPostgres.ApiPostgresRepo) apiController {
	if controller.service == nil {
		controller = apiController{
			service: factoryApiService(repo),
		}
	}
	return controller
}
