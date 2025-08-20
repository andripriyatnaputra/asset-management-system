package models

type Employee struct {
	ID             int64   `json:"id"`
	EmployeeNIK    string  `json:"employee_nik"`
	Name           string  `json:"name"`
	Email          string  `json:"email"`
	DepartmentID   *int64  `json:"department_id"`             // Pointer untuk handle NULL
	DepartmentName *string `json:"department_name,omitempty"` // Untuk menampilkan nama saat join

	Role string `json:"role"`

	Password     string `json:"-"`
	PasswordHash string `json:"-"`
}
