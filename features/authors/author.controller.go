package authors

import (
	"encoding/json"
	"fmt"
	"net/http"
	models "schoolify_be/models"
	repoPostgres "schoolify_be/repositories/postgresql"
	utilities "schoolify_be/utilities"
	custom_errors "schoolify_be/utilities/errors"
	errUtility "schoolify_be/utilities/errors"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
)

type authorController struct {
	service authorService
}

var controller authorController

func (a authorController) GetAllAuthorPostgresCtrl(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	query := r.URL.Query()
	var param models.ParamGetListAuthor
	search := query.Get("search")
	param.Search = search
	limit := query.Get("limit")
	limitNum, err := strconv.Atoi(limit)
	if err != nil {
		limitNum = 10
	}
	param.Limit = uint16(limitNum)
	offset := query.Get("offset")
	offsetNum, err := strconv.Atoi(offset)
	if err != nil {
		offsetNum = 0
	}
	param.Offset = uint16(offsetNum)
	sortField := query.Get("sortField")
	param.SortField = sortField
	sortOrder := query.Get("sortOrder")
	param.SortOrder = sortOrder
	validator := validator.New()
	err = validator.Struct(param)
	if err != nil {
		utilities.Log(utilities.ERROR, r.URL.Path, "GetAllAuthorPostgresCtrl", nil, err.Error(), nil)
		w.WriteHeader(http.StatusBadRequest)
		objError := custom_errors.ParseError(err)
		strError, _ := json.Marshal(objError)
		fmt.Fprintf(w, `{"errors": %s}`, strError)
		return
	}
	listAuthor, count, customErr := a.service.getAllAuthorPostgresSrvc(param)
	if customErr != (errUtility.CustomError{}) {
		customErr.Compile()
		utilities.Log(utilities.ERROR, r.URL.Path, "GetAllAuthorPostgresCtrl", param, customErr.Message, nil)
		w.WriteHeader(int(customErr.Code))
		fmt.Fprintf(w, `{"errors": "%s"}`, customErr.MessageToSend)
		return
	}
	if listAuthor == nil {
		listAuthor = []models.Author{}
	}
	strRes, err := json.Marshal(map[string]any{
		"data":    listAuthor,
		"count":   count,
		"message": "Success get list author",
	})
	if err != nil {
		utilities.Log(utilities.ERROR, r.URL.Path, "GetAllAuthorPostgresCtrl", nil, err.Error(), nil)
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, `{"message":"Internal Server Error"}`)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write(strRes)
}
func (a authorController) GetAuthorByIDPostgresCtrl(w http.ResponseWriter, r *http.Request) {
	authorID := chi.URLParam(r, "id")
	w.Header().Set("Content-Type", "application/json")
	if authorID == "" {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, `{"message":"Invalid Author ID"}`)
		return
	}
	author, errObj := a.service.getAuthorByIDPostgresSrvc(authorID)
	if errObj.Code != 0 {
		errObj.Compile()
		utilities.Log(utilities.ERROR, r.URL.Path, "GetAuthorByIDPostgresCtrl", map[string]string{"id": authorID}, errObj.Message, errObj)
		w.WriteHeader(int(errObj.Code))
		fmt.Fprintf(w, `{"message":"%s"}`, errObj.MessageToSend)
		return
	}
	strData, err := json.Marshal(map[string]any{
		"data":    author,
		"message": "Success get an author by id",
	})
	if err != nil {
		utilities.Log(utilities.ERROR, r.URL.Path, "GetAuthorByIDPostgresCtrl", nil, err.Error(), nil)
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, `{"message":"Internal Server Error"}`)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write(strData)
}
func (a authorController) CreateAuthorPostgresCtrl(w http.ResponseWriter, r *http.Request) {
	var param models.CreateAuthorRequest
	w.Header().Set("Content-Type", "application/json")

	err := json.NewDecoder(r.Body).Decode(&param)
	if err != nil {
		utilities.Log(utilities.WARN, r.URL.Path, "CreateAuthorPostgresCtrl", r.Body, "Invalid JSON format: "+err.Error(), nil)
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, `{"message":"Invalid Data"}`)
		return
	}

	val := validator.New()
	err = val.Struct(param)
	if err != nil {
		objError := errUtility.ParseError(err)
		strError, _ := json.Marshal(objError)
		// Warning log for validation error
		utilities.Log(utilities.WARN, r.URL.Path, "CreateAuthorPostgresCtrl", param, "Validation error: "+string(strError), objError)
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, `{"errors":%s}`, strError)
		return
	}
	isErr, errObj := a.service.createAuthorPostgresSrvc(param)
	if isErr {
		errObj.Compile()
		// Error log for service error
		utilities.Log(utilities.ERROR, r.URL.Path, "CreateAuthorPostgresCtrl", param, "Service error: "+errObj.Message, errObj)
		w.WriteHeader(int(errObj.Code))
		fmt.Fprintf(w, `{"message":"%s"}`, errObj.MessageToSend)
		return
	}
	w.WriteHeader(http.StatusCreated)
	fmt.Fprintf(w, `{"message":"Author created successfully"}`)

}

func factoryAuthorController(repo repoPostgres.AuthorPostgresRepo) authorController {
	if controller == (authorController{}) {
		controller = authorController{
			service: factoryAuthorService(repo),
		}
	}
	return controller
}
