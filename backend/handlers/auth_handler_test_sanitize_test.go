package handlers

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSanitizeInitSQLForTests(t *testing.T) {
	input := `CREATE TABLE public.foo (id int);
ALTER TABLE public.foo OWNER TO admin;
GRANT ALL ON TABLE public.foo TO admin;
REVOKE ALL ON TABLE public.foo FROM public;
SET SESSION AUTHORIZATION 'admin';
ALTER DEFAULT PRIVILEGES FOR ROLE admin IN SCHEMA public GRANT ALL ON TABLES TO admin;
INSERT INTO public.foo VALUES (1);`

	output := sanitizeInitSQLForTests(input)

	assert.Contains(t, output, "CREATE TABLE public.foo (id int);")
	assert.Contains(t, output, "INSERT INTO public.foo VALUES (1);")
	assert.NotContains(t, output, "OWNER TO")
	assert.NotContains(t, output, "GRANT ALL")
	assert.NotContains(t, output, "REVOKE ALL")
	assert.NotContains(t, output, "SET SESSION AUTHORIZATION")
	assert.NotContains(t, output, "ALTER DEFAULT PRIVILEGES")
}
