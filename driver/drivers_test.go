package driver

import (
	"testing"

	mocks "github.com/hashicorp/consul-terraform-sync/mocks/templates"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestDrivers_Add(t *testing.T) {
	cases := []struct {
		name      string
		taskName  string
		driver    Driver
		expectErr bool
	}{
		{
			"happy path",
			"happy_task",
			&Terraform{},
			false,
		},
		{
			"error: empty task name",
			"",
			&Terraform{},
			true,
		},
		{
			"error: nil driver",
			"nil_driver",
			nil,
			true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			drivers := NewDrivers()
			err := drivers.Add(tc.taskName, tc.driver)
			if tc.expectErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				actual, ok := drivers.drivers[tc.taskName]
				require.True(t, ok)
				assert.Equal(t, tc.driver, actual)
			}
		})
	}

	t.Run("driver already exists", func(t *testing.T) {
		taskName := "already_exists"
		drivers := NewDrivers()
		err := drivers.Add(taskName, &Terraform{})
		require.NoError(t, err)

		err = drivers.Add(taskName, &Terraform{})
		require.Error(t, err)
	})
}

func TestDrivers_Get(t *testing.T) {
	cases := []struct {
		name     string
		taskName string
		ok       bool
	}{
		{
			"driver exists",
			"task_a",
			true,
		},
		{
			"driver doesn't exist",
			"non_existent_task",
			false,
		},
	}

	drivers := NewDrivers()
	err := drivers.Add("task_a", &Terraform{})
	require.NoError(t, err)

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d, ok := drivers.Get(tc.taskName)
			require.Equal(t, tc.ok, ok)
			if tc.ok {
				assert.NotNil(t, d)
			} else {
				assert.Nil(t, d)
			}
		})
	}
}

func TestDrivers_Map(t *testing.T) {
	t.Run("drivers map", func(t *testing.T) {
		drivers := NewDrivers()
		err := drivers.Add("task_a", &Terraform{})
		require.NoError(t, err)
		err = drivers.Add("task_b", &Terraform{})
		require.NoError(t, err)

		m := drivers.Map()

		// check that expected tasks are in returned map
		_, ok := m["task_a"]
		require.True(t, ok)
		_, ok = m["task_b"]
		require.True(t, ok)

		// check that adding a driver does not modify returned map
		err = drivers.Add("task_c", &Terraform{})
		require.NoError(t, err)
		_, ok = m["task_c"]
		require.False(t, ok)
	})
}

func TestDrivers_Delete(t *testing.T) {
	cases := []struct {
		name      string
		taskName  string
		expectErr bool
	}{
		{
			"driver_exists",
			"task_a",
			false,
		},
		{
			"driver_does_not_exist",
			"non_existent_task",
			false,
		},
		{
			"empty_task_name",
			"",
			true,
		},
	}

	drivers := NewDrivers()

	w := new(mocks.Watcher)
	w.On("Deregister", mock.Anything).Return()

	err := drivers.Add("task_a", &Terraform{watcher: w})
	require.NoError(t, err)

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := drivers.Delete(tc.taskName)
			if tc.expectErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				_, ok := drivers.Get(tc.taskName)
				assert.False(t, ok)
			}
		})
	}
}

func TestDrivers_MarkForDeletion(t *testing.T) {
	drivers := NewDrivers()
	name := "test_task"
	drivers.MarkForDeletion(name)
	assert.True(t, drivers.deletion[name])

	drivers.deletion[name] = false
	drivers.MarkForDeletion(name)
	assert.True(t, drivers.deletion[name])
}

func TestDrivers_IsMarkedForDeletion(t *testing.T) {
	name := "test_task"

	t.Run("true", func(t *testing.T) {
		drivers := NewDrivers()
		drivers.deletion[name] = true
		assert.True(t, drivers.IsMarkedForDeletion(name))
	})

	t.Run("false", func(t *testing.T) {
		drivers := NewDrivers()
		drivers.deletion[name] = false
		assert.False(t, drivers.IsMarkedForDeletion(name))
	})

	t.Run("does_not_exist", func(t *testing.T) {
		drivers := NewDrivers()
		assert.False(t, drivers.IsMarkedForDeletion(name))
	})
}
