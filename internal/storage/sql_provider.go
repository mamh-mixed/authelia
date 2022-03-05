package storage

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"

	"github.com/authelia/authelia/v4/internal/authentication"
	"github.com/authelia/authelia/v4/internal/configuration/schema"
	"github.com/authelia/authelia/v4/internal/logging"
	"github.com/authelia/authelia/v4/internal/model"
)

// NewSQLProvider generates a generic SQLProvider to be used with other SQL provider NewUp's.
func NewSQLProvider(config *schema.Configuration, name, driverName, dataSourceName string) (provider SQLProvider) {
	db, err := sqlx.Open(driverName, dataSourceName)

	provider = SQLProvider{
		db:         db,
		key:        sha256.Sum256([]byte(config.Storage.EncryptionKey)),
		name:       name,
		driverName: driverName,
		config:     config,
		errOpen:    err,
		log:        logging.Logger(),

		sqlInsertAuthenticationAttempt:            fmt.Sprintf(queryFmtInsertAuthenticationLogEntry, tableAuthenticationLogs),
		sqlSelectAuthenticationAttemptsByUsername: fmt.Sprintf(queryFmtSelect1FAAuthenticationLogEntryByUsername, tableAuthenticationLogs),

		sqlInsertIdentityVerification:  fmt.Sprintf(queryFmtInsertIdentityVerification, tableIdentityVerification),
		sqlConsumeIdentityVerification: fmt.Sprintf(queryFmtConsumeIdentityVerification, tableIdentityVerification),
		sqlSelectIdentityVerification:  fmt.Sprintf(queryFmtSelectIdentityVerification, tableIdentityVerification),

		sqlUpsertTOTPConfig:  fmt.Sprintf(queryFmtUpsertTOTPConfiguration, tableTOTPConfigurations),
		sqlDeleteTOTPConfig:  fmt.Sprintf(queryFmtDeleteTOTPConfiguration, tableTOTPConfigurations),
		sqlSelectTOTPConfig:  fmt.Sprintf(queryFmtSelectTOTPConfiguration, tableTOTPConfigurations),
		sqlSelectTOTPConfigs: fmt.Sprintf(queryFmtSelectTOTPConfigurations, tableTOTPConfigurations),

		sqlUpdateTOTPConfigSecret:                 fmt.Sprintf(queryFmtUpdateTOTPConfigurationSecret, tableTOTPConfigurations),
		sqlUpdateTOTPConfigSecretByUsername:       fmt.Sprintf(queryFmtUpdateTOTPConfigurationSecretByUsername, tableTOTPConfigurations),
		sqlUpdateTOTPConfigRecordSignIn:           fmt.Sprintf(queryFmtUpdateTOTPConfigRecordSignIn, tableTOTPConfigurations),
		sqlUpdateTOTPConfigRecordSignInByUsername: fmt.Sprintf(queryFmtUpdateTOTPConfigRecordSignInByUsername, tableTOTPConfigurations),

		sqlUpsertWebauthnDevice:            fmt.Sprintf(queryFmtUpsertWebauthnDevice, tableWebauthnDevices),
		sqlSelectWebauthnDevices:           fmt.Sprintf(queryFmtSelectWebauthnDevices, tableWebauthnDevices),
		sqlSelectWebauthnDevicesByUsername: fmt.Sprintf(queryFmtSelectWebauthnDevicesByUsername, tableWebauthnDevices),

		sqlUpdateWebauthnDevicePublicKey:              fmt.Sprintf(queryFmtUpdateWebauthnDevicePublicKey, tableWebauthnDevices),
		sqlUpdateWebauthnDevicePublicKeyByUsername:    fmt.Sprintf(queryFmtUpdateUpdateWebauthnDevicePublicKeyByUsername, tableWebauthnDevices),
		sqlUpdateWebauthnDeviceRecordSignIn:           fmt.Sprintf(queryFmtUpdateWebauthnDeviceRecordSignIn, tableWebauthnDevices),
		sqlUpdateWebauthnDeviceRecordSignInByUsername: fmt.Sprintf(queryFmtUpdateWebauthnDeviceRecordSignInByUsername, tableWebauthnDevices),

		sqlUpsertDuoDevice: fmt.Sprintf(queryFmtUpsertDuoDevice, tableDuoDevices),
		sqlDeleteDuoDevice: fmt.Sprintf(queryFmtDeleteDuoDevice, tableDuoDevices),
		sqlSelectDuoDevice: fmt.Sprintf(queryFmtSelectDuoDevice, tableDuoDevices),

		sqlUpsertPreferred2FAMethod: fmt.Sprintf(queryFmtUpsertPreferred2FAMethod, tableUserPreferences),
		sqlSelectPreferred2FAMethod: fmt.Sprintf(queryFmtSelectPreferred2FAMethod, tableUserPreferences),
		sqlSelectUserInfo:           fmt.Sprintf(queryFmtSelectUserInfo, tableTOTPConfigurations, tableWebauthnDevices, tableDuoDevices, tableUserPreferences),

		// Table: oauth2_authorize_code_sessions.
		sqlInsertOAuth2AuthorizeCodeSession:            fmt.Sprintf(queryFmtInsertOAuth2Session, tableOAuth2AuthorizeCodeSessions),
		sqlSelectOAuth2AuthorizeCodeSession:            fmt.Sprintf(queryFmtSelectOAuth2Session, tableOAuth2AuthorizeCodeSessions),
		sqlRevokeOAuth2AuthorizeCodeSession:            fmt.Sprintf(queryFmtRevokeOAuth2Session, tableOAuth2AuthorizeCodeSessions),
		sqlRevokeOAuth2AuthorizeCodeSessionByRequestID: fmt.Sprintf(queryFmtRevokeOAuth2SessionByRequestID, tableOAuth2AuthorizeCodeSessions),

		// Table: oauth2_access_token_sessions.
		sqlInsertOAuth2AccessTokenSession:            fmt.Sprintf(queryFmtInsertOAuth2Session, tableOAuth2AccessTokenSessions),
		sqlSelectOAuth2AccessTokenSession:            fmt.Sprintf(queryFmtSelectOAuth2Session, tableOAuth2AccessTokenSessions),
		sqlRevokeOAuth2AccessTokenSession:            fmt.Sprintf(queryFmtRevokeOAuth2Session, tableOAuth2AccessTokenSessions),
		sqlRevokeOAuth2AccessTokenSessionByRequestID: fmt.Sprintf(queryFmtRevokeOAuth2SessionByRequestID, tableOAuth2AccessTokenSessions),

		// Table: oauth2_refresh_token_sessions.
		sqlInsertOAuth2RefreshTokenSession:            fmt.Sprintf(queryFmtInsertOAuth2Session, tableOAuth2RefreshTokenSessions),
		sqlSelectOAuth2RefreshTokenSession:            fmt.Sprintf(queryFmtSelectOAuth2Session, tableOAuth2RefreshTokenSessions),
		sqlRevokeOAuth2RefreshTokenSession:            fmt.Sprintf(queryFmtRevokeOAuth2Session, tableOAuth2RefreshTokenSessions),
		sqlRevokeOAuth2RefreshTokenSessionByRequestID: fmt.Sprintf(queryFmtRevokeOAuth2SessionByRequestID, tableOAuth2RefreshTokenSessions),

		// Table: oauth2_pkce_request_sessions.
		sqlInsertOAuth2PKCERequestSession:            fmt.Sprintf(queryFmtInsertOAuth2Session, tableOAuth2PKCERequestSessions),
		sqlSelectOAuth2PKCERequestSession:            fmt.Sprintf(queryFmtSelectOAuth2Session, tableOAuth2PKCERequestSessions),
		sqlRevokeOAuth2PKCERequestSession:            fmt.Sprintf(queryFmtRevokeOAuth2Session, tableOAuth2PKCERequestSessions),
		sqlRevokeOAuth2PKCERequestSessionByRequestID: fmt.Sprintf(queryFmtRevokeOAuth2SessionByRequestID, tableOAuth2PKCERequestSessions),

		// Table: oauth2_openid_connect_sessions.
		sqlInsertOAuth2OpenIDConnectSession:            fmt.Sprintf(queryFmtInsertOAuth2Session, tableOAuth2OpenIDConnectSessions),
		sqlSelectOAuth2OpenIDConnectSession:            fmt.Sprintf(queryFmtSelectOAuth2Session, tableOAuth2OpenIDConnectSessions),
		sqlRevokeOAuth2OpenIDConnectSession:            fmt.Sprintf(queryFmtRevokeOAuth2Session, tableOAuth2OpenIDConnectSessions),
		sqlRevokeOAuth2OpenIDConnectSessionByRequestID: fmt.Sprintf(queryFmtRevokeOAuth2SessionByRequestID, tableOAuth2OpenIDConnectSessions),

		// Table: oauth2_blacklisted_jti.
		sqlUpsertOAuth2BlacklistedJTI: fmt.Sprintf(queryFmtUpsertOAuth2BlacklistedJTI, tableOAuth2BlacklistedJTI),
		sqlSelectOAuth2BlacklistedJTI: fmt.Sprintf(queryFmtSelectOAuth2BlacklistedJTI, tableOAuth2BlacklistedJTI),

		sqlInsertMigration:       fmt.Sprintf(queryFmtInsertMigration, tableMigrations),
		sqlSelectMigrations:      fmt.Sprintf(queryFmtSelectMigrations, tableMigrations),
		sqlSelectLatestMigration: fmt.Sprintf(queryFmtSelectLatestMigration, tableMigrations),

		sqlUpsertEncryptionValue: fmt.Sprintf(queryFmtUpsertEncryptionValue, tableEncryption),
		sqlSelectEncryptionValue: fmt.Sprintf(queryFmtSelectEncryptionValue, tableEncryption),

		sqlFmtRenameTable: queryFmtRenameTable,
	}

	return provider
}

// SQLProvider is a storage provider persisting data in a SQL database.
type SQLProvider struct {
	db         *sqlx.DB
	key        [32]byte
	name       string
	driverName string
	schema     string
	config     *schema.Configuration
	errOpen    error

	log *logrus.Logger

	// Table: authentication_logs.
	sqlInsertAuthenticationAttempt            string
	sqlSelectAuthenticationAttemptsByUsername string

	// Table: identity_verification.
	sqlInsertIdentityVerification  string
	sqlConsumeIdentityVerification string
	sqlSelectIdentityVerification  string

	// Table: totp_configurations.
	sqlUpsertTOTPConfig  string
	sqlDeleteTOTPConfig  string
	sqlSelectTOTPConfig  string
	sqlSelectTOTPConfigs string

	sqlUpdateTOTPConfigSecret                 string
	sqlUpdateTOTPConfigSecretByUsername       string
	sqlUpdateTOTPConfigRecordSignIn           string
	sqlUpdateTOTPConfigRecordSignInByUsername string

	// Table: webauthn_devices.
	sqlUpsertWebauthnDevice            string
	sqlSelectWebauthnDevices           string
	sqlSelectWebauthnDevicesByUsername string

	sqlUpdateWebauthnDevicePublicKey              string
	sqlUpdateWebauthnDevicePublicKeyByUsername    string
	sqlUpdateWebauthnDeviceRecordSignIn           string
	sqlUpdateWebauthnDeviceRecordSignInByUsername string

	// Table: duo_devices.
	sqlUpsertDuoDevice string
	sqlDeleteDuoDevice string
	sqlSelectDuoDevice string

	// Table: user_preferences.
	sqlUpsertPreferred2FAMethod string
	sqlSelectPreferred2FAMethod string
	sqlSelectUserInfo           string

	// Table: migrations.
	sqlInsertMigration       string
	sqlSelectMigrations      string
	sqlSelectLatestMigration string

	// Table: encryption.
	sqlUpsertEncryptionValue string
	sqlSelectEncryptionValue string

	// Table: oauth2_authorize_code_sessions.
	sqlInsertOAuth2AuthorizeCodeSession            string
	sqlSelectOAuth2AuthorizeCodeSession            string
	sqlRevokeOAuth2AuthorizeCodeSession            string
	sqlRevokeOAuth2AuthorizeCodeSessionByRequestID string

	// Table: oauth2_access_token_sessions.
	sqlInsertOAuth2AccessTokenSession            string
	sqlSelectOAuth2AccessTokenSession            string
	sqlRevokeOAuth2AccessTokenSession            string
	sqlRevokeOAuth2AccessTokenSessionByRequestID string

	// Table: oauth2_refresh_token_sessions.
	sqlInsertOAuth2RefreshTokenSession            string
	sqlSelectOAuth2RefreshTokenSession            string
	sqlRevokeOAuth2RefreshTokenSession            string
	sqlRevokeOAuth2RefreshTokenSessionByRequestID string

	// Table: oauth2_pkce_request_sessions.
	sqlInsertOAuth2PKCERequestSession            string
	sqlSelectOAuth2PKCERequestSession            string
	sqlRevokeOAuth2PKCERequestSession            string
	sqlRevokeOAuth2PKCERequestSessionByRequestID string

	// Table: oauth2_openid_connect_sessions.
	sqlInsertOAuth2OpenIDConnectSession            string
	sqlSelectOAuth2OpenIDConnectSession            string
	sqlRevokeOAuth2OpenIDConnectSession            string
	sqlRevokeOAuth2OpenIDConnectSessionByRequestID string

	sqlUpsertOAuth2BlacklistedJTI string
	sqlSelectOAuth2BlacklistedJTI string

	// Utility.
	sqlSelectExistingTables string
	sqlFmtRenameTable       string
}

// Close the underlying database connection.
func (p *SQLProvider) Close() (err error) {
	return p.db.Close()
}

// StartupCheck implements the provider startup check interface.
func (p *SQLProvider) StartupCheck() (err error) {
	if p.errOpen != nil {
		return fmt.Errorf("error opening database: %w", p.errOpen)
	}

	// TODO: Decide if this is needed, or if it should be configurable.
	for i := 0; i < 19; i++ {
		if err = p.db.Ping(); err == nil {
			break
		}

		time.Sleep(time.Millisecond * 500)
	}

	if err != nil {
		return fmt.Errorf("error pinging database: %w", err)
	}

	p.log.Infof("Storage schema is being checked for updates")

	ctx := context.Background()

	if err = p.SchemaEncryptionCheckKey(ctx, false); err != nil && !errors.Is(err, ErrSchemaEncryptionVersionUnsupported) {
		return err
	}

	err = p.SchemaMigrate(ctx, true, SchemaLatest)

	switch err {
	case ErrSchemaAlreadyUpToDate:
		p.log.Infof("Storage schema is already up to date")
		return nil
	case nil:
		return nil
	default:
		return fmt.Errorf("error during schema migrate: %w", err)
	}
}

// BeginTX begins a transaction.
func (p *SQLProvider) BeginTX(ctx context.Context) (c context.Context, err error) {
	var tx *sql.Tx

	if tx, err = p.db.Begin(); err != nil {
		return nil, err
	}

	return context.WithValue(ctx, ctxKeyTransaction, tx), nil
}

// Commit performs a database commit.
func (p *SQLProvider) Commit(ctx context.Context) (err error) {
	tx, ok := ctx.Value(ctxKeyTransaction).(*sql.Tx)

	if !ok {
		return errors.New("could not retrieve tx")
	}

	return tx.Commit()
}

// Rollback performs a database rollback.
func (p *SQLProvider) Rollback(ctx context.Context) (err error) {
	tx, ok := ctx.Value(ctxKeyTransaction).(*sql.Tx)

	if !ok {
		return errors.New("could not retrieve tx")
	}

	return tx.Rollback()
}

// SaveOAuth2Session saves a OAuth2Session to the database.
func (p *SQLProvider) SaveOAuth2Session(ctx context.Context, sessionType OAuth2SessionType, session *model.OAuth2Session) (err error) {
	var query string

	switch sessionType {
	case OAuth2SessionTypeAuthorizeCode:
		query = p.sqlInsertOAuth2AuthorizeCodeSession
	case OAuth2SessionTypeAccessToken:
		query = p.sqlInsertOAuth2AccessTokenSession
	case OAuth2SessionTypeRefreshToken:
		query = p.sqlInsertOAuth2RefreshTokenSession
	case OAuth2SessionTypePKCEChallenge:
		query = p.sqlInsertOAuth2PKCERequestSession
	case OAuth2SessionTypeOpenIDConnect:
		query = p.sqlInsertOAuth2OpenIDConnectSession
	default:
		return fmt.Errorf("error inserting oauth2 session for subject '%s' and request id '%s': unknown oauth2 session type '%s'", session.Subject, session.RequestID, sessionType)
	}

	if session.Session, err = p.encrypt(session.Session); err != nil {
		return fmt.Errorf("error encrypting the oauth2 %s session data for subject '%s' and request id '%s': %w", session.Subject, session.RequestID, sessionType, err)
	}

	_, err = p.db.ExecContext(ctx, query,
		session.RequestID, session.ClientID, session.Signature,
		session.Subject, session.RequestedAt, session.RequestedScopes, session.GrantedScopes,
		session.RequestedAudience, session.GrantedAudience, session.Form, session.Session)

	if err != nil {
		return fmt.Errorf("error inserting oauth2 %s session data for subject '%s' and request id '%s': %w", session.Subject, session.RequestID, sessionType, err)
	}

	return nil
}

// RevokeOAuth2Session marks a OAuth2Session as revoked in the database.
func (p *SQLProvider) RevokeOAuth2Session(ctx context.Context, sessionType OAuth2SessionType, signature string) (err error) {
	var query string

	switch sessionType {
	case OAuth2SessionTypeAuthorizeCode:
		query = p.sqlRevokeOAuth2AuthorizeCodeSession
	case OAuth2SessionTypeAccessToken:
		query = p.sqlRevokeOAuth2AccessTokenSession
	case OAuth2SessionTypeRefreshToken:
		query = p.sqlRevokeOAuth2RefreshTokenSession
	case OAuth2SessionTypePKCEChallenge:
		query = p.sqlRevokeOAuth2PKCERequestSession
	case OAuth2SessionTypeOpenIDConnect:
		query = p.sqlRevokeOAuth2OpenIDConnectSession
	default:
		return fmt.Errorf("error revoking oauth2 session with signature '%s': unknown oauth2 session type '%s'", signature, sessionType)
	}

	if _, err = p.db.ExecContext(ctx, query, signature); err != nil {
		return fmt.Errorf("error revoking oauth2 %s session with signature '%s': %w", sessionType, signature, err)
	}

	return nil
}

// RevokeOAuth2SessionByRequestID marks a OAuth2Session as revoked in the database.
func (p *SQLProvider) RevokeOAuth2SessionByRequestID(ctx context.Context, sessionType OAuth2SessionType, requestID string) (err error) {
	var query string

	switch sessionType {
	case OAuth2SessionTypeAuthorizeCode:
		query = p.sqlRevokeOAuth2AuthorizeCodeSessionByRequestID
	case OAuth2SessionTypeAccessToken:
		query = p.sqlRevokeOAuth2AccessTokenSessionByRequestID
	case OAuth2SessionTypeRefreshToken:
		query = p.sqlRevokeOAuth2RefreshTokenSessionByRequestID
	case OAuth2SessionTypePKCEChallenge:
		query = p.sqlRevokeOAuth2PKCERequestSessionByRequestID
	case OAuth2SessionTypeOpenIDConnect:
		query = p.sqlRevokeOAuth2OpenIDConnectSessionByRequestID
	default:
		return fmt.Errorf("error revoking oauth2 session with request id '%s': unknown oauth2 session type '%s'", requestID, sessionType)
	}

	if _, err = p.db.ExecContext(ctx, query, requestID); err != nil {
		return fmt.Errorf("error revoking oauth2 %s session with request id '%s': %w", sessionType, requestID, err)
	}

	return nil
}

// LoadOAuth2Session saves a OAuth2Session from the database.
func (p *SQLProvider) LoadOAuth2Session(ctx context.Context, sessionType OAuth2SessionType, signature string) (session *model.OAuth2Session, err error) {
	var query string

	switch sessionType {
	case OAuth2SessionTypeAuthorizeCode:
		query = p.sqlSelectOAuth2AuthorizeCodeSession
	case OAuth2SessionTypeAccessToken:
		query = p.sqlSelectOAuth2AccessTokenSession
	case OAuth2SessionTypeRefreshToken:
		query = p.sqlSelectOAuth2RefreshTokenSession
	case OAuth2SessionTypePKCEChallenge:
		query = p.sqlSelectOAuth2PKCERequestSession
	case OAuth2SessionTypeOpenIDConnect:
		query = p.sqlSelectOAuth2OpenIDConnectSession
	default:
		return nil, fmt.Errorf("error selecting oauth2 session: unknown oauth2 session type '%s'", sessionType)
	}

	session = &model.OAuth2Session{}

	if err = p.db.GetContext(ctx, &session, query, signature); err != nil {
		return nil, fmt.Errorf("error selecting oauth2 %s session: %w", sessionType, err)
	}

	return session, nil
}

// SaveOAuth2BlacklistedJTI saves a OAuth2BlacklistedJTI to the database.
func (p *SQLProvider) SaveOAuth2BlacklistedJTI(ctx context.Context, blacklistedJTI *model.OAuth2BlacklistedJTI) (err error) {
	if _, err = p.db.ExecContext(ctx, p.sqlUpsertOAuth2BlacklistedJTI, blacklistedJTI.Signature, blacklistedJTI.ExpiresAt); err != nil {
		return fmt.Errorf("error inserting oauth2 blacklisted JTI with signature '%s': %w", blacklistedJTI.Signature, err)
	}

	return nil
}

// LoadOAuth2BlacklistedJTI loads a OAuth2BlacklistedJTI from the database.
func (p *SQLProvider) LoadOAuth2BlacklistedJTI(ctx context.Context, signature string) (blacklistedJTI *model.OAuth2BlacklistedJTI, err error) {
	blacklistedJTI = &model.OAuth2BlacklistedJTI{}

	if err = p.db.GetContext(ctx, blacklistedJTI, p.sqlSelectOAuth2BlacklistedJTI, signature); err != nil {
		return nil, err
	}

	return blacklistedJTI, nil
}

// SavePreferred2FAMethod save the preferred method for 2FA to the database.
func (p *SQLProvider) SavePreferred2FAMethod(ctx context.Context, username string, method string) (err error) {
	if _, err = p.db.ExecContext(ctx, p.sqlUpsertPreferred2FAMethod, username, method); err != nil {
		return fmt.Errorf("error upserting preferred two factor method for user '%s': %w", username, err)
	}

	return nil
}

// LoadPreferred2FAMethod load the preferred method for 2FA from the database.
func (p *SQLProvider) LoadPreferred2FAMethod(ctx context.Context, username string) (method string, err error) {
	err = p.db.GetContext(ctx, &method, p.sqlSelectPreferred2FAMethod, username)

	switch {
	case err == nil:
		return method, nil
	case errors.Is(err, sql.ErrNoRows):
		return "", nil
	default:
		return "", fmt.Errorf("error selecting preferred two factor method for user '%s': %w", username, err)
	}
}

// LoadUserInfo loads the model.UserInfo from the database.
func (p *SQLProvider) LoadUserInfo(ctx context.Context, username string) (info model.UserInfo, err error) {
	err = p.db.GetContext(ctx, &info, p.sqlSelectUserInfo, username, username, username, username)

	switch {
	case err == nil:
		return info, nil
	case errors.Is(err, sql.ErrNoRows):
		if _, err = p.db.ExecContext(ctx, p.sqlUpsertPreferred2FAMethod, username, authentication.PossibleMethods[0]); err != nil {
			return model.UserInfo{}, fmt.Errorf("error upserting preferred two factor method while selecting user info for user '%s': %w", username, err)
		}

		if err = p.db.GetContext(ctx, &info, p.sqlSelectUserInfo, username, username, username, username); err != nil {
			return model.UserInfo{}, fmt.Errorf("error selecting user info for user '%s': %w", username, err)
		}

		return info, nil
	default:
		return model.UserInfo{}, fmt.Errorf("error selecting user info for user '%s': %w", username, err)
	}
}

// SaveIdentityVerification save an identity verification record to the database.
func (p *SQLProvider) SaveIdentityVerification(ctx context.Context, verification model.IdentityVerification) (err error) {
	if _, err = p.db.ExecContext(ctx, p.sqlInsertIdentityVerification,
		verification.JTI, verification.IssuedAt, verification.IssuedIP, verification.ExpiresAt,
		verification.Username, verification.Action); err != nil {
		return fmt.Errorf("error inserting identity verification for user '%s' with uuid '%s': %w", verification.Username, verification.JTI, err)
	}

	return nil
}

// ConsumeIdentityVerification marks an identity verification record in the database as consumed.
func (p *SQLProvider) ConsumeIdentityVerification(ctx context.Context, jti string, ip model.NullIP) (err error) {
	if _, err = p.db.ExecContext(ctx, p.sqlConsumeIdentityVerification, ip, jti); err != nil {
		return fmt.Errorf("error updating identity verification: %w", err)
	}

	return nil
}

// FindIdentityVerification checks if an identity verification record is in the database and active.
func (p *SQLProvider) FindIdentityVerification(ctx context.Context, jti string) (found bool, err error) {
	verification := model.IdentityVerification{}
	if err = p.db.GetContext(ctx, &verification, p.sqlSelectIdentityVerification, jti); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}

		return false, fmt.Errorf("error selecting identity verification exists: %w", err)
	}

	switch {
	case verification.Consumed != nil:
		return false, fmt.Errorf("the token has already been consumed")
	case verification.ExpiresAt.Before(time.Now()):
		return false, fmt.Errorf("the token expired %s ago", time.Since(verification.ExpiresAt))
	default:
		return true, nil
	}
}

// SaveTOTPConfiguration save a TOTP configuration of a given user in the database.
func (p *SQLProvider) SaveTOTPConfiguration(ctx context.Context, config model.TOTPConfiguration) (err error) {
	if config.Secret, err = p.encrypt(config.Secret); err != nil {
		return fmt.Errorf("error encrypting the TOTP configuration secret for user '%s': %w", config.Username, err)
	}

	if _, err = p.db.ExecContext(ctx, p.sqlUpsertTOTPConfig,
		config.CreatedAt, config.LastUsedAt,
		config.Username, config.Issuer,
		config.Algorithm, config.Digits, config.Period, config.Secret); err != nil {
		return fmt.Errorf("error upserting TOTP configuration for user '%s': %w", config.Username, err)
	}

	return nil
}

// UpdateTOTPConfigurationSignIn updates a registered Webauthn devices sign in information.
func (p *SQLProvider) UpdateTOTPConfigurationSignIn(ctx context.Context, id int, lastUsedAt *time.Time) (err error) {
	if _, err = p.db.ExecContext(ctx, p.sqlUpdateTOTPConfigRecordSignIn, lastUsedAt, id); err != nil {
		return fmt.Errorf("error updating TOTP configuration id %d: %w", id, err)
	}

	return nil
}

// DeleteTOTPConfiguration delete a TOTP configuration from the database given a username.
func (p *SQLProvider) DeleteTOTPConfiguration(ctx context.Context, username string) (err error) {
	if _, err = p.db.ExecContext(ctx, p.sqlDeleteTOTPConfig, username); err != nil {
		return fmt.Errorf("error deleting TOTP configuration for user '%s': %w", username, err)
	}

	return nil
}

// LoadTOTPConfiguration load a TOTP configuration given a username from the database.
func (p *SQLProvider) LoadTOTPConfiguration(ctx context.Context, username string) (config *model.TOTPConfiguration, err error) {
	config = &model.TOTPConfiguration{}

	if err = p.db.QueryRowxContext(ctx, p.sqlSelectTOTPConfig, username).StructScan(config); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNoTOTPConfiguration
		}

		return nil, fmt.Errorf("error selecting TOTP configuration for user '%s': %w", username, err)
	}

	if config.Secret, err = p.decrypt(config.Secret); err != nil {
		return nil, fmt.Errorf("error decrypting the TOTP secret for user '%s': %w", username, err)
	}

	return config, nil
}

// LoadTOTPConfigurations load a set of TOTP configurations.
func (p *SQLProvider) LoadTOTPConfigurations(ctx context.Context, limit, page int) (configs []model.TOTPConfiguration, err error) {
	configs = make([]model.TOTPConfiguration, 0, limit)

	if err = p.db.SelectContext(ctx, &configs, p.sqlSelectTOTPConfigs, limit, limit*page); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}

		return nil, fmt.Errorf("error selecting TOTP configurations: %w", err)
	}

	for i, c := range configs {
		if configs[i].Secret, err = p.decrypt(c.Secret); err != nil {
			return nil, fmt.Errorf("error decrypting TOTP configuration for user '%s': %w", c.Username, err)
		}
	}

	return configs, nil
}

func (p *SQLProvider) updateTOTPConfigurationSecret(ctx context.Context, config model.TOTPConfiguration) (err error) {
	switch config.ID {
	case 0:
		_, err = p.db.ExecContext(ctx, p.sqlUpdateTOTPConfigSecretByUsername, config.Secret, config.Username)
	default:
		_, err = p.db.ExecContext(ctx, p.sqlUpdateTOTPConfigSecret, config.Secret, config.ID)
	}

	if err != nil {
		return fmt.Errorf("error updating TOTP configuration secret for user '%s': %w", config.Username, err)
	}

	return nil
}

// SaveWebauthnDevice saves a registered Webauthn device.
func (p *SQLProvider) SaveWebauthnDevice(ctx context.Context, device model.WebauthnDevice) (err error) {
	if device.PublicKey, err = p.encrypt(device.PublicKey); err != nil {
		return fmt.Errorf("error encrypting the Webauthn device public key for user '%s' kid '%x': %w", device.Username, device.KID, err)
	}

	if _, err = p.db.ExecContext(ctx, p.sqlUpsertWebauthnDevice,
		device.CreatedAt, device.LastUsedAt,
		device.RPID, device.Username, device.Description,
		device.KID, device.PublicKey,
		device.AttestationType, device.Transport, device.AAGUID, device.SignCount, device.CloneWarning,
	); err != nil {
		return fmt.Errorf("error upserting Webauthn device for user '%s' kid '%x': %w", device.Username, device.KID, err)
	}

	return nil
}

// UpdateWebauthnDeviceSignIn updates a registered Webauthn devices sign in information.
func (p *SQLProvider) UpdateWebauthnDeviceSignIn(ctx context.Context, id int, rpid string, lastUsedAt *time.Time, signCount uint32, cloneWarning bool) (err error) {
	if _, err = p.db.ExecContext(ctx, p.sqlUpdateWebauthnDeviceRecordSignIn, rpid, lastUsedAt, signCount, cloneWarning, id); err != nil {
		return fmt.Errorf("error updating Webauthn signin metadata for id '%x': %w", id, err)
	}

	return nil
}

// LoadWebauthnDevices loads Webauthn device registrations.
func (p *SQLProvider) LoadWebauthnDevices(ctx context.Context, limit, page int) (devices []model.WebauthnDevice, err error) {
	devices = make([]model.WebauthnDevice, 0, limit)

	if err = p.db.SelectContext(ctx, &devices, p.sqlSelectWebauthnDevices, limit, limit*page); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}

		return nil, fmt.Errorf("error selecting Webauthn devices: %w", err)
	}

	for i, device := range devices {
		if devices[i].PublicKey, err = p.decrypt(device.PublicKey); err != nil {
			return nil, fmt.Errorf("error decrypting Webauthn public key for user '%s': %w", device.Username, err)
		}
	}

	return devices, nil
}

// LoadWebauthnDevicesByUsername loads all webauthn devices registration for a given username.
func (p *SQLProvider) LoadWebauthnDevicesByUsername(ctx context.Context, username string) (devices []model.WebauthnDevice, err error) {
	if err = p.db.SelectContext(ctx, &devices, p.sqlSelectWebauthnDevicesByUsername, username); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNoWebauthnDevice
		}

		return nil, fmt.Errorf("error selecting Webauthn devices for user '%s': %w", username, err)
	}

	for i, device := range devices {
		if devices[i].PublicKey, err = p.decrypt(device.PublicKey); err != nil {
			return nil, fmt.Errorf("error decrypting Webauthn public key for user '%s': %w", username, err)
		}
	}

	return devices, nil
}

func (p *SQLProvider) updateWebauthnDevicePublicKey(ctx context.Context, device model.WebauthnDevice) (err error) {
	switch device.ID {
	case 0:
		_, err = p.db.ExecContext(ctx, p.sqlUpdateWebauthnDevicePublicKeyByUsername, device.PublicKey, device.Username, device.KID)
	default:
		_, err = p.db.ExecContext(ctx, p.sqlUpdateWebauthnDevicePublicKey, device.PublicKey, device.ID)
	}

	if err != nil {
		return fmt.Errorf("error updating Webauthn public key for user '%s' kid '%x': %w", device.Username, device.KID, err)
	}

	return nil
}

// SavePreferredDuoDevice saves a Duo device.
func (p *SQLProvider) SavePreferredDuoDevice(ctx context.Context, device model.DuoDevice) (err error) {
	if _, err = p.db.ExecContext(ctx, p.sqlUpsertDuoDevice, device.Username, device.Device, device.Method); err != nil {
		return fmt.Errorf("error upserting preferred duo device for user '%s': %w", device.Username, err)
	}

	return nil
}

// DeletePreferredDuoDevice deletes a Duo device of a given user.
func (p *SQLProvider) DeletePreferredDuoDevice(ctx context.Context, username string) (err error) {
	if _, err = p.db.ExecContext(ctx, p.sqlDeleteDuoDevice, username); err != nil {
		return fmt.Errorf("error deleting preferred duo device for user '%s': %w", username, err)
	}

	return nil
}

// LoadPreferredDuoDevice loads a Duo device of a given user.
func (p *SQLProvider) LoadPreferredDuoDevice(ctx context.Context, username string) (device *model.DuoDevice, err error) {
	device = &model.DuoDevice{}

	if err = p.db.QueryRowxContext(ctx, p.sqlSelectDuoDevice, username).StructScan(device); err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNoDuoDevice
		}

		return nil, fmt.Errorf("error selecting preferred duo device for user '%s': %w", username, err)
	}

	return device, nil
}

// AppendAuthenticationLog append a mark to the authentication log.
func (p *SQLProvider) AppendAuthenticationLog(ctx context.Context, attempt model.AuthenticationAttempt) (err error) {
	if _, err = p.db.ExecContext(ctx, p.sqlInsertAuthenticationAttempt,
		attempt.Time, attempt.Successful, attempt.Banned, attempt.Username,
		attempt.Type, attempt.RemoteIP, attempt.RequestURI, attempt.RequestMethod); err != nil {
		return fmt.Errorf("error inserting authentication attempt for user '%s': %w", attempt.Username, err)
	}

	return nil
}

// LoadAuthenticationLogs retrieve the latest failed authentications from the authentication log.
func (p *SQLProvider) LoadAuthenticationLogs(ctx context.Context, username string, fromDate time.Time, limit, page int) (attempts []model.AuthenticationAttempt, err error) {
	attempts = make([]model.AuthenticationAttempt, 0, limit)

	if err = p.db.SelectContext(ctx, &attempts, p.sqlSelectAuthenticationAttemptsByUsername, fromDate, username, limit, limit*page); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNoAuthenticationLogs
		}

		return nil, fmt.Errorf("error selecting authentication logs for user '%s': %w", username, err)
	}

	return attempts, nil
}
