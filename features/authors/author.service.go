package authors

import (
	"net/http"
	models "schoolify_be/models"
	repoPostgres "schoolify_be/repositories/postgresql"
	utilities "schoolify_be/utilities"
	errUtility "schoolify_be/utilities/errors"
	"sync"
)

type authorService struct {
	repo repoPostgres.AuthorPostgresRepo
}

var service authorService

func (s authorService) createAuthorPostgresSrvc(param models.CreateAuthorRequest) (bool, errUtility.CustomError) {
	uuid, err := utilities.GenerateUUIDv7()
	if err != nil {
		return true, errUtility.CustomError{
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		}
	}
	param.UUID = uuid
	return s.repo.CreateAuthor(param)
}
func (s authorService) getAllAuthorPostgresSrvc(param models.ParamGetListAuthor) ([]models.Author, int, errUtility.CustomError) {
	var (
		listAuthor []models.Author
		count      int
		errList    errUtility.CustomError
		errCount   errUtility.CustomError
		wg         sync.WaitGroup
	)

	wg.Add(2)

	// goroutine count
	go func() {
		defer wg.Done()
		count, errCount = s.repo.GetCountAuthor(param.Search)
	}()

	// goroutine list
	go func() {
		defer wg.Done()
		listAuthor, errList = s.repo.GetListAuthor(param)
	}()

	wg.Wait()

	if errCount.Code != 0 {
		return nil, 0, errCount
	}

	if errList.Code != 0 {
		return nil, 0, errList
	}

	return listAuthor, count, errUtility.CustomError{}
}
func (s authorService) getAuthorByIDPostgresSrvc(authorID string) (models.Author, errUtility.CustomError) {
	return s.repo.GetAuthorByID(authorID)
}
func factoryAuthorService(repo repoPostgres.AuthorPostgresRepo) authorService {
	if service == (authorService{}) {
		service = authorService{
			repo,
		}
	}
	return service
}
