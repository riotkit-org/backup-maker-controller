package domain

const InternalTemplateKind = "internal"

// InternalTemplate
// ----------------
//
// Does NOT provide script and image.
// The script is provided by Backup Maker inside container
// so... we DO NOT DO anything on the operator level, we let run Backup Maker
// and use its own internal, bundled templates
//

type InternalTemplate struct {
	Name string
}

func (it InternalTemplate) GetImage() string {
	return ""
}

func (it InternalTemplate) GetBackupScript() string {
	return ""
}

func (it InternalTemplate) GetRestoreScript() string {
	return ""
}

func (it InternalTemplate) ProvidesScript() bool {
	return false
}

func (it InternalTemplate) GetName() string {
	return it.Name
}
