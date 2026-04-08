package submission

import (
	"errors"
	"main/database/pgsql"
	"main/model"
	_const "main/util/const"
	"main/util/password"
	"main/util/token"
	"testing"
)

func TestRegisterAuthorSrcBranches(t *testing.T) {
	setupSubmissionRouteMocks(t)

	getAuthorByAuthorNameFn = func(author *model.Author) error {
		return errors.New("db down")
	}
	if err := registerAuthorSrc(&model.Author{AuthorName: "a"}); err == nil {
		t.Fatal("register should fail on unexpected lookup error")
	}

	getAuthorByAuthorNameFn = func(author *model.Author) error { return nil }
	if err := registerAuthorSrc(&model.Author{AuthorName: "a"}); err == nil {
		t.Fatal("register should fail when username exists")
	}

	getAuthorByAuthorNameFn = func(author *model.Author) error { return _const.UsernameNotExist }
	createAuthorFn = func(author *model.Author) error {
		if author.AuthorName != "new" {
			t.Fatalf("unexpected author name: %s", author.AuthorName)
		}
		return nil
	}
	if err := registerAuthorSrc(&model.Author{AuthorName: "new"}); err != nil {
		t.Fatalf("register should succeed: %v", err)
	}
}

func TestAuthorLoginSrcAndRefreshTokenSrc(t *testing.T) {
	setupSubmissionRouteMocks(t)

	hash, err := password.HashPassword("secret")
	if err != nil {
		t.Fatalf("hash password failed: %v", err)
	}

	getAuthorByAuthorIDFn = func(author *model.Author) error {
		author.AuthorID = 8
		author.Password = hash
		return nil
	}

	resp, err := authorLoginSrc(&model.Author{AuthorID: 8, Password: "secret"})
	if err != nil {
		t.Fatalf("authorLoginSrc should succeed: %v", err)
	}
	claims, err := token.ParseAccessToken(resp.Token)
	if err != nil {
		t.Fatalf("parse access token failed: %v", err)
	}
	if claims.Role != "author" {
		t.Fatalf("expected author role in access token, got %s", claims.Role)
	}

	_, err = authorLoginSrc(&model.Author{AuthorID: 8, Password: "bad"})
	if err == nil {
		t.Fatal("authorLoginSrc should fail for bad password")
	}

	refresh, err := refreshTokenSrc(8)
	if err != nil {
		t.Fatalf("refreshTokenSrc should succeed: %v", err)
	}
	refreshClaims, err := token.ParseAccessToken(refresh.Token)
	if err != nil {
		t.Fatalf("parse refresh access token failed: %v", err)
	}
	if refreshClaims.Role != "author" {
		t.Fatalf("refresh token role should be author, got %s", refreshClaims.Role)
	}
}

func TestUpdateAndFindSubmissionSrc(t *testing.T) {
	setupSubmissionRouteMocks(t)

	updated := false
	updateAuthorFn = func(author *model.Author) error {
		updated = true
		return nil
	}
	if err := updateAuthorSrc(&model.Author{AuthorID: 1, AuthorName: "n"}); err != nil {
		t.Fatalf("updateAuthorSrc should succeed: %v", err)
	}
	if !updated {
		t.Fatal("updateAuthorFn should be called")
	}

	updateWorkFn = func(work *model.Work) error { return nil }
	setUploadFilePermissionFn = func(authorID int, trackID int, workID int) error { return nil }
	getStartAndEndDateFn = func(trackID int) (int64, int64, error) {
		return 0, 4102444800, nil
	}
	resolveFlowForExecutionFn = func(scope string, eventKey string, targetType string, targetID int) (model.ScriptFlow, []pgsql.ResolvedFlowStep, error) {
		return model.ScriptFlow{}, nil, pgsql.ErrFlowNotMounted
	}
	if err := updateSubmissionSrc(&model.Work{WorkID: 1, AuthorID: 1, TrackID: 2, WorkTitle: "x"}); err != nil {
		t.Fatalf("updateSubmissionSrc should succeed: %v", err)
	}

	findWorksByAuthorIDFn = func(authorID int) ([]model.Work, error) {
		return []model.Work{{WorkID: 1, AuthorID: authorID}}, nil
	}
	works, err := findSubmissionsByAuthorIDSrc(3)
	if err != nil {
		t.Fatalf("findSubmissionsByAuthorIDSrc should succeed: %v", err)
	}
	if len(works) != 1 || works[0].AuthorID != 3 {
		t.Fatalf("unexpected works: %+v", works)
	}
}
