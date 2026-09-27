package postgresql

import (
	"database/sql"
	"net/http"
	connections "schoolify_be/connections"
	models "schoolify_be/models"
	utility "schoolify_be/utilities"
	errUtility "schoolify_be/utilities/errors"
	"strings"
)

type AuthorPostgresRepo struct{}

var repo AuthorPostgresRepo

func (a AuthorPostgresRepo) CreateAuthor(param models.CreateAuthorRequest) (bool, errUtility.CustomError) {
	_, err := connections.DbPostgres().Query("INSERT INTO public.authors(id, name) VALUES($1,$2);", param.UUID, param.Name)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate key value violates unique constraint") {

			return true, errUtility.CustomError{
				Code:          http.StatusUnprocessableEntity,
				Message:       err.Error(),
				MessageToSend: "Duplicate data",
				Function:      "AuthorPostgresRepo.CreateAuthor",
			}
		}
		return true, errUtility.CustomError{
			Code:     http.StatusInternalServerError,
			Message:  err.Error(),
			Function: "AuthorPostgresRepo.CreateAuthor",
		}
	}
	return false, errUtility.CustomError{}
}
func (AuthorPostgresRepo) GetListAuthor(param models.ParamGetListAuthor) ([]models.Author, errUtility.CustomError) {
	var listAuthor []models.Author
	var nullableUpdatedAt sql.NullTime
	results, err := connections.DbPostgres().Query("SELECT id, name, created, updated FROM public.get_list_author($1, $2, $3, $4, $5);", param.SortField, param.SortOrder, param.Limit, param.Offset, param.Search)
	if err != nil {
		return []models.Author{}, errUtility.CustomError{
			Code:          http.StatusInternalServerError,
			Message:       err.Error(),
			MessageToSend: "Internal Server Error",
			Function:      "AuthorPostgresRepo.GetListAuthor",
		}
	}
	for results.Next() {
		var data models.Author
		err := results.Scan(&data.ID, &data.Name, &data.Created, &nullableUpdatedAt)
		if err != nil {
			return []models.Author{}, errUtility.CustomError{
				Code:          http.StatusInternalServerError,
				Message:       err.Error(),
				MessageToSend: "Internal Server Error",
				Function:      "AuthorPostgresRepo.GetAuthorByID",
			}
		}
		if nullableUpdatedAt.Valid {
			data.Updated = nullableUpdatedAt.Time
			data.StrUpdated = utility.FormatTimeMillis(nullableUpdatedAt.Time)
		} else {
			data.Updated = data.Created
			data.StrUpdated = utility.FormatTimeMillis(data.Created)
		}
		listAuthor = append(listAuthor, data)
	}
	return listAuthor, errUtility.CustomError{}
}
func (AuthorPostgresRepo) GetAuthorByID(authorID string) (models.Author, errUtility.CustomError) {
	var author models.Author
	var nullableUpdatedAt sql.NullTime
	err := connections.DbPostgres().QueryRow("SELECT id, name, created, updated FROM public.authors WHERE id=$1;", authorID).Scan(&author.ID, &author.Name, &author.Created, &nullableUpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return models.Author{}, errUtility.CustomError{
				Code:          http.StatusNotFound,
				Message:       "Author not found",
				MessageToSend: "Author not found",
				Function:      "AuthorPostgresRepo.GetAuthorByID",
			}
		} else if strings.Contains(err.Error(), "invalid input syntax for type uuid") {
			return models.Author{}, errUtility.CustomError{
				Code:          http.StatusNotFound,
				Message:       "Invalid UUID",
				MessageToSend: "Author not found",
				Function:      "AuthorPostgresRepo.GetAuthorByID",
			}
		}

		return models.Author{}, errUtility.CustomError{
			Code:     http.StatusInternalServerError,
			Message:  err.Error(),
			Function: "AuthorPostgresRepo.GetAuthorByID",
		}
	}
	author.StrCreated = utility.FormatTimeMillis(author.Created)
	if nullableUpdatedAt.Valid {
		author.Updated = nullableUpdatedAt.Time
		author.StrUpdated = utility.FormatTimeMillis(nullableUpdatedAt.Time)
	} else {
		author.Updated = author.Created
		author.StrUpdated = utility.FormatTimeMillis(author.Created)
	}
	return author, errUtility.CustomError{}
}
func (AuthorPostgresRepo) GetCountAuthor(search string) (int, errUtility.CustomError) {
	var count int
	err := connections.DbPostgres().QueryRow("SELECT get_count_author FROM public.get_count_author($1);", search).Scan(&count)
	if err != nil {
		return 0, errUtility.CustomError{
			Code:          http.StatusInternalServerError,
			Message:       err.Error(),
			MessageToSend: "Internal Server Error",
			Function:      "AuthorPostgresRepo.GetCountAuthor",
		}
	}
	return count, errUtility.CustomError{}
}
func FactoryAuthorPostgresRepo() AuthorPostgresRepo {
	if repo == (AuthorPostgresRepo{}) {
		repo = AuthorPostgresRepo{}
	}
	return repo
}
