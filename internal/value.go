package internal

import (
	"encoding/json"
	"math/big"
	"reflect"
	"regexp"
	"strconv"
	"time"

	"github.com/tada/catch"
	"github.com/tada/dgo/dgo"
)

var reflectValueType = reflect.TypeOf((*dgo.Value)(nil)).Elem()

// New creates an instance of the given type from the given argument
func New(typ dgo.Type, argument dgo.Value) dgo.Value {
	if nt, ok := typ.(dgo.Factory); ok {
		return nt.New(argument)
	}
	if args, ok := argument.(dgo.Arguments); ok {
		if args.Len() != 1 {
			panic(catch.Error(`unable to create a %v from arguments %v`, typ, args))
		}
		argument = args.Get(0)
	}
	if typ.Instance(argument) {
		return argument
	}
	panic(catch.Error(`unable to create a %v from %v`, typ, argument))
}

// Value returns the immutable dgo.Value representation of its argument. If the argument type
// is known, it will be more efficient to use explicit methods such as Float(), String(),
// Map(), etc.
func Value(v interface{}) dgo.Value {
	// This goFunc is kept very small to enable inlining so this
	// if statement should not be baked in to the grand switch
	// in the value goFunc
	if gv, ok := v.(dgo.Value); ok {
		return gv
	}
	if dv := value(v, false); dv != nil {
		return dv
	}
	return ValueFromReflected(reflect.ValueOf(v), false)
}

func value(v interface{}, frozen bool) dgo.Value {
	var dv dgo.Value
	switch v := v.(type) {
	case nil:
		dv = Nil
	case string:
		dv = String(v)
	case int:
		dv = Int64(int64(v))
	case bool:
		dv = boolean(v)
	case []byte:
		dv = Binary(v, frozen)
	case []string:
		dv = Strings(v)
	case []int:
		dv = Integers(v)
	case *regexp.Regexp:
		dv = Regexp(v)
	case time.Time:
		dv = &timeVal{v}
	case *time.Time:
		dv = &timeVal{*v}
	case *big.Int:
		dv = BigInt(v)
	case *big.Float:
		dv = BigFloat(v)
	case uint:
		dv = unsignedToInteger(uint64(v))
	case uint64:
		dv = unsignedToInteger(v)
	case error:
		dv = &errw{v}
	case json.Number:
		dv = FromJSONNumber(v)
	case reflect.Value:
		dv = ValueFromReflected(v, frozen)
	case reflect.Type:
		dv = TypeFromReflected(v)
	default:
		if i, ok := ToInt(v); ok {
			dv = intVal(i)
		} else {
			var f float64
			if f, ok = ToFloat(v); ok {
				dv = floatVal(f)
			}
		}
	}
	return dv
}

// FromJSONNumber converts the given json.Number to a dgo.Number
func FromJSONNumber(v json.Number) dgo.Number {
	var nv dgo.Number
	i, err := v.Int64()
	if err == nil {
		nv = Int64(i)
	} else if numErr, ok := err.(*strconv.NumError); ok && numErr.Err == strconv.ErrRange {
		var u uint64
		if u, err = strconv.ParseUint(v.String(), 0, 64); err == nil {
			nv = Uint64(u)
		}
	}
	if err != nil {
		var f float64
		if f, err = v.Float64(); err == nil {
			nv = Float(f)
		} else {
			panic(catch.Error(err))
		}
	}
	return nv
}

// ValueFromReflected converts the given reflected value into an immutable dgo.Value
func ValueFromReflected(vr reflect.Value, frozen bool) dgo.Value {
	// Invalid shouldn't happen, but needs a check
	if !vr.IsValid() {
		return Nil
	}

	isPtr := false
	switch vr.Kind() {
	case reflect.Slice:
		return ArrayFromReflected(vr, frozen)
	case reflect.Map:
		return FromReflectedMap(vr, frozen)
	case reflect.Interface:
		if vr.Type().NumMethod() == 0 {
			return ValueFromReflected(vr.Elem(), frozen)
		}
	case reflect.Ptr:
		if vr.IsNil() {
			return Nil
		}
		isPtr = true
	case reflect.Func:
		return (*goFunc)(&vr)
	}

	if vr.CanInterface() {
		vi := vr.Interface()
		if v, ok := vi.(dgo.Value); ok {
			return v
		}
		if v := value(vi, frozen); v != nil {
			return v
		}
	}

	if isPtr {
		er := vr.Elem()
		// Pointer to struct should have been handled at this point or it is a pointer to
		// an unknown struct and should be a native
		if er.Kind() != reflect.Struct {
			return ValueFromReflected(er, frozen)
		}
	}
	// Value as unsafe. Immutability is not guaranteed
	return Native(vr)
}

// FromValue converts a dgo.Value into a go native value. The given `dest` must be a pointer
// to the expected native value.
func FromValue(src dgo.Value, dest interface{}) {
	dp := reflect.ValueOf(dest)
	if reflect.Ptr != dp.Kind() {
		panic(catch.Error("destination is not a pointer"))
	}
	ReflectTo(src, dp.Elem())
}

// ReflectTo assigns the given dgo.Value to the given reflect.Value
func ReflectTo(src dgo.Value, dest reflect.Value) {
	if !dest.Type().AssignableTo(reflectValueType) {
		if rv, ok := src.(dgo.ReflectedValue); ok {
			rv.ReflectTo(dest)
			return
		}
	}
	dest.Set(reflect.ValueOf(src))
}

// Add well known types like regexp, time, etc. here
var wellKnownTypes = map[reflect.Type]dgo.Type{
	reflect.TypeOf(&regexp.Regexp{}): DefaultRegexpType,
	reflect.TypeOf(time.Time{}):      DefaultTimeType,
}
