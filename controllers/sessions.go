package controllers

import (
	"auth/config"
	"auth/enums"
	"auth/functools"
	"auth/infrastructure"
	"auth/inout"
	"auth/models"
	"context"
	uuid "github.com/satori/go.uuid"
	"time"
)

func getUserFromTokens(store infrastructure.StoreInterface, controller infrastructure.LoginControllerInterface, ctx context.Context, accessToken string, fingerprint string, refreshToken string) (int, *uuid.UUID) {

	status, payload := controller.Login(ctx, accessToken)
	if status != enums.Ok {
		return status, nil
	}

	if store.GetSession(ctx, fingerprint, refreshToken, payload.UserID) == nil {
		return enums.SessionNotFound, nil
	}

	return enums.Ok, &payload.UserID
}

func getUserFromEmailAndCode(store infrastructure.StoreInterface, ctx context.Context, emailValue string, emailCode string) (int, *uuid.UUID) {

	emailValue = functools.NormalizeEmail(emailValue)
	if emailValue == "" {
		return enums.IncorrectEmail, nil
	}

	code := store.GetEmailConfirmationCode(ctx, emailValue)
	if code == "" {
		return enums.EmailConfirmationCodeNotFound, nil
	} else if code != emailCode {
		return enums.IncorrectEmailCode, nil
	}

	status, email := store.GetEmail(ctx, emailValue)
	if status != enums.Ok {
		return status, nil
	}
	if email == nil {
		return enums.EmailNotFound, nil
	}

	return enums.Ok, &email.UserId
}

func getUserFromEmailAndPassword(store infrastructure.StoreInterface, controller infrastructure.LoginControllerInterface, ctx context.Context, emailValue string, passwordValue string) (int, *uuid.UUID) {

	emailValue = functools.NormalizeEmail(emailValue)
	if emailValue == "" {
		return enums.IncorrectEmail, nil
	}

	status, email := store.GetEmail(ctx, emailValue)
	if status != enums.Ok {
		return status, nil
	}
	if email == nil {
		return enums.EmailNotFound, nil
	}

	status, password := store.GetLatestPassword(ctx, email.UserId)
	if status != enums.Ok {
		return status, nil
	}
	if password == nil {
		return enums.PasswordNotFound, nil
	}

	passwordVerified := controller.VerifyPassword(ctx, passwordValue, password.Value)
	if !passwordVerified {
		return enums.IncorrectPassword, nil
	}

	return enums.Ok, &password.UserId
}

func getUserFromPhoneAndPassword(store infrastructure.StoreInterface, controller infrastructure.LoginControllerInterface, ctx context.Context, phoneValue string, passwordValue string, phoneCountryCode string) (int, *uuid.UUID) {
	phoneValue = functools.NormalizePhone(phoneValue, phoneCountryCode)
	if phoneValue == "" {
		return enums.IncorrectPhone, nil
	}

	status, phone := store.GetPhone(ctx, phoneValue)
	if status != enums.Ok {
		return status, nil
	}
	if phone == nil {
		return enums.PhoneNotFound, nil
	}

	status, password := store.GetLatestPassword(ctx, phone.UserId)
	if status != enums.Ok {
		return status, nil
	}
	if password == nil {
		return enums.PasswordNotFound, nil
	}

	passwordVerified := controller.VerifyPassword(ctx, passwordValue, password.Value)
	if !passwordVerified {
		return enums.IncorrectPassword, nil
	}

	return enums.Ok, &password.UserId
}

func getUserFromPhoneAndCode(store infrastructure.StoreInterface, ctx context.Context, phoneValue string, phoneCode string, phoneCountryCode string) (int, *uuid.UUID) {
	phoneValue = functools.NormalizePhone(phoneValue, phoneCountryCode)
	if phoneValue == "" {
		return enums.IncorrectPhone, nil
	}

	code := store.GetPhoneConfirmationCode(ctx, phoneValue)
	if code == "" {
		return enums.PhoneConfirmationCodeNotFound, nil
	} else if code != phoneCode {
		return enums.IncorrectPhoneCode, nil
	}

	status, phone := store.GetPhone(ctx, phoneValue)
	if status != enums.Ok {
		return status, nil
	}
	if phone == nil {
		return enums.PhoneNotFound, nil
	}

	return enums.Ok, &phone.UserId
}

func getUserByCredentials(store infrastructure.StoreInterface, controller infrastructure.LoginControllerInterface, ctx context.Context, credentials inout.CreateSessionRequestV1) (int, *uuid.UUID) {
	var status int
	var userID *uuid.UUID

	switch credentials.Data.(type) {
	case *inout.CreateSessionRequestV1_Tokens_:
		tokens := credentials.GetTokens()
		status, userID = getUserFromTokens(store, controller, ctx, tokens.AccessToken, credentials.Fingerprint, tokens.RefreshToken)
	case *inout.CreateSessionRequestV1_EmailAndPassword_:
		emailAndPassword := credentials.GetEmailAndPassword()
		status, userID = getUserFromEmailAndPassword(store, controller, ctx, emailAndPassword.Email, emailAndPassword.Password)
	case *inout.CreateSessionRequestV1_EmailAndCode_:
		emailAndCode := credentials.GetEmailAndCode()
		status, userID = getUserFromEmailAndCode(store, ctx, emailAndCode.Email, emailAndCode.Code)
	case *inout.CreateSessionRequestV1_PhoneAndPassword_:
		phoneAndPassword := credentials.GetPhoneAndPassword()
		status, userID = getUserFromPhoneAndPassword(store, controller, ctx, phoneAndPassword.Phone, phoneAndPassword.Password, phoneAndPassword.PhoneCountryCode)
	case *inout.CreateSessionRequestV1_PhoneAndCode_:
		phoneAndCode := credentials.GetPhoneAndCode()
		status, userID = getUserFromPhoneAndCode(store, ctx, phoneAndCode.Phone, phoneAndCode.Code, phoneAndCode.PhoneCountryCode)
	default:
		return enums.CredentialsNotProvided, nil
	}

	return status, userID
}

func CreateSession(
	store infrastructure.StoreInterface,
	loginController infrastructure.LoginControllerInterface,
	ctx context.Context,
	credentials inout.CreateSessionRequestV1) (int, *models.Session) {
	status, userID := getUserByCredentials(store, loginController, ctx, credentials)
	secret := store.GetActualSecret(ctx)
	status, session := store.CreateSession(ctx, credentials.Fingerprint, *userID, secret.Id, credentials.UserAgent)
	if status != enums.Ok {
		return status, nil
	}
	userView := store.GetUserView(ctx, *userID)
	if userView == nil {
		return enums.UserNotFound, nil
	}
	env := config.GetEnvironment()
	expires := time.Now().Add(time.Minute*time.Duration(env.AccessTokenLifetime))
	session.Expires = expires.Unix()
	session.AccessToken = loginController.EncodeAccessToken(ctx, *userID, userView.Roles, secret, expires)

	return status, session
}
