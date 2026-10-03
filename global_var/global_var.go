package global_var

type DatabaseConnection struct {
	Driver       string // "mysql" atau "postgres"
	Host         string
	Port         string
	User         string
	Password     string
	DatabaseName string
	CreateDBTest bool
}

type ResponseFormat struct {
	Status  int         `json:"status"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

type UserRole string

const (
	RoleAdmin    UserRole = "admin"
	RoleManager  UserRole = "manager" 
	RoleEmployee UserRole = "employee"
)

var (
	DB          interface{}
	JWTSecret   string
	Environment string
)