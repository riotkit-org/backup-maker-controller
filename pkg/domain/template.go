package domain

type Template interface {
	GetImage() string
	GetBackupScript() string
	GetRestoreScript() string
	ProvidesScript() bool
	GetName() string
}
