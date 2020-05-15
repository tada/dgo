package internal_test

import (
	"testing"

	"github.com/tada/dgo/dgo"
	"github.com/tada/dgo/test/assert"
	"github.com/tada/dgo/tf"
	"github.com/tada/dgo/typ"
	"github.com/tada/dgo/vf"
)

func TestIllegalAssignment(t *testing.T) {
	v := tf.IllegalAssignment(typ.String, vf.Int64(3))
	assert.Equal(t, v, tf.IllegalAssignment(typ.String, vf.Int64(3)))
	assert.NotEqual(t, v, tf.IllegalAssignment(typ.String, vf.Int64(4)))
	assert.NotEqual(t, v, `oops`)
	assert.Equal(t, `the value 3 cannot be assigned to a variable of type string`, v.Error())
}

func TestIllegalAssignment_nil(t *testing.T) {
	v := tf.IllegalAssignment(typ.String, nil)
	assert.Equal(t, `nil cannot be assigned to a variable of type string`, v.Error())
}

func TestIllegalSize(t *testing.T) {
	v := tf.IllegalSize(tf.String(1, 10), 12)
	assert.Equal(t, v, tf.IllegalSize(tf.String(1, 10), 12))
	assert.NotEqual(t, v, tf.IllegalSize(tf.String(1, 10), 11))
	assert.NotEqual(t, v, `oops`)
	assert.Equal(t, `size constraint violation on type string[1,10] when attempting resize to 12`, v.Error())
}

func TestIllegalMapKey(t *testing.T) {
	tp := tf.ParseType(`{a:string}`).(dgo.StructMapType)
	v := tf.IllegalMapKey(tp, vf.String(`b`))
	assert.Equal(t, v, tf.IllegalMapKey(tp, vf.String(`b`)))
	assert.NotEqual(t, v, tf.IllegalMapKey(tp, vf.String(`c`)))
	assert.NotEqual(t, v, `oops`)
	assert.Equal(t, `key "b" cannot be added to type {"a":string}`, v.Error())
}
