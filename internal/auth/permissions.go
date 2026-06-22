package auth

type Permission string

const (
	PermUsersManage     Permission = "users:manage"
	PermShowsRead       Permission = "shows:read"
	PermShowsWrite      Permission = "shows:write"
	PermShowsPublish    Permission = "shows:publish"
	PermEpisodesRead    Permission = "episodes:read"
	PermEpisodesWrite   Permission = "episodes:write"
	PermEpisodesPublish Permission = "episodes:publish"
	PermImportsRun      Permission = "imports:run"
)

type Role string

const (
	RoleAdmin  Role = "admin"
	RoleEditor Role = "editor"
	RoleViewer Role = "viewer"
)

var rolePermissions = map[Role][]Permission{
	RoleAdmin: {
		PermUsersManage,
		PermShowsRead, PermShowsWrite, PermShowsPublish,
		PermEpisodesRead, PermEpisodesWrite, PermEpisodesPublish,
		PermImportsRun,
	},
	RoleEditor: {
		PermShowsRead, PermShowsWrite, PermShowsPublish,
		PermEpisodesRead, PermEpisodesWrite, PermEpisodesPublish,
		PermImportsRun,
	},
	RoleViewer: {
		PermShowsRead, PermEpisodesRead,
	},
}

func (r Role) IsValid() bool {
	_, ok := rolePermissions[r]
	return ok
}

func PermissionsForRole(r Role) []Permission { return rolePermissions[r] }
