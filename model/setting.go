package model

import (
	"encoding/json/v2"
	"fmt"

	"kr.dev/errorfmt"

	"ily.dev/act3/database/schema"
)

// Setting key constants.
const (
	SettingKeyTransmissionBaseURL = "transmission.base-url"
	SettingKeyTMDBAccessToken     = "tmdb.access-token"
)

// SettingType is the Go type of a setting.
type SettingType string

const (
	SettingTypeInt    SettingType = "int"
	SettingTypeString SettingType = "string"
	SettingTypeBool   SettingType = "bool"
)

type settingDef struct {
	Group   string
	Type    SettingType
	Default *string // JSON-encoded default; nil means zero value for Type
}

// settingDefs maps each valid key to its type, group, and default.
var settingDefs = map[string]settingDef{
	SettingKeyTransmissionBaseURL: {Group: "transmission", Type: SettingTypeString},
	SettingKeyTMDBAccessToken:     {Group: "tmdb", Type: SettingTypeString},
}

// SettingDefaultInt sets the default value for an int setting.
// It panics if the key is not defined or its type is not int.
//
// All calls to SettingDefaultInt must happen before any calls to
// TxR.SettingGetByGroup or TxRW.SettingSet.
func SettingDefaultInt(key string, n int) {
	setDefault(key, SettingTypeInt, n)
}

// SettingDefaultString sets the default value for a string setting.
// It panics if the key is not defined or its type is not string.
//
// All calls to SettingDefaultString must happen before any calls to
// TxR.SettingGetByGroup or TxRW.SettingSet.
func SettingDefaultString(key string, v string) {
	setDefault(key, SettingTypeString, v)
}

// SettingDefaultBool sets the default value for a bool setting.
// It panics if the key is not defined or its type is not bool.
//
// All calls to SettingDefaultBool must happen before any calls to
// TxR.SettingGetByGroup or TxRW.SettingSet.
func SettingDefaultBool(key string, v bool) {
	setDefault(key, SettingTypeBool, v)
}

func setDefault(key string, typ SettingType, val any) {
	def, ok := settingDefs[key]
	if !ok {
		panic(fmt.Sprintf("setting %s not defined", key))
	}
	if def.Type != typ {
		panic(fmt.Sprintf("setting %s: type is %s, not %s", key, def.Type, typ))
	}
	def.Default = new(mustMarshalJSON(val))
	settingDefs[key] = def
}

// settingZeroJSON maps each SettingType to the JSON encoding
// of its zero value.
var settingZeroJSON = map[SettingType]string{
	SettingTypeInt:    "0",
	SettingTypeString: `""`,
	SettingTypeBool:   "false",
}

func (def settingDef) defaultJSON() string {
	if def.Default != nil {
		return *def.Default
	}
	return settingZeroJSON[def.Type]
}

// settingHooks maps setting keys to callbacks invoked when the setting
// is written via SettingSet.
// Hooks are not called for defaults or during startup config loading.
var settingHooks = map[string][]func(*Setting){}

// SettingHook registers a callback to be called whenever
// the setting with the given key is updated via SettingSet.
func SettingHook(key string, f func(*Setting)) {
	if _, ok := settingDefs[key]; !ok {
		panic(fmt.Sprintf("setting %s not defined", key))
	}
	settingHooks[key] = append(settingHooks[key], f)
}

// Settings is a map of setting key to Setting.
type Settings map[string]*Setting

// Setting is a key-value pair with a JSON-encoded value.
type Setting struct {
	s schema.Setting
}

// Key returns the setting's unique key.
func (s *Setting) Key() string { return s.s.Key }

// Group returns the group s belongs to.
func (s *Setting) Group() string { return s.s.Group }

// Value returns the value of s as JSON.
func (s *Setting) Value() string { return s.s.Value }

// Type returns the registered type of s.
func (s *Setting) Type() SettingType { return settingDefs[s.s.Key].Type }

// Int returns the int value of s.
//
// It panics if the registered type of s is not int.
func (s *Setting) Int() int {
	s.mustType(SettingTypeInt)
	var n int
	err := json.Unmarshal([]byte(s.s.Value), &n)
	if err != nil {
		panic(fmt.Sprintf("setting %s: unmarshal int: %v", s.s.Key, err))
	}
	return n
}

// String returns the string value of s.
//
// It panics if the registered type of s is not string.
func (s *Setting) String() string {
	s.mustType(SettingTypeString)
	var v string
	err := json.Unmarshal([]byte(s.s.Value), &v)
	if err != nil {
		panic(fmt.Sprintf("setting %s: unmarshal string: %v", s.s.Key, err))
	}
	return v
}

// Bool returns the bool value of s.
//
// It panics if the registered type of s is not bool.
func (s *Setting) Bool() bool {
	s.mustType(SettingTypeBool)
	var v bool
	err := json.Unmarshal([]byte(s.s.Value), &v)
	if err != nil {
		panic(fmt.Sprintf("setting %s: unmarshal bool: %v", s.s.Key, err))
	}
	return v
}

func (s *Setting) mustType(want SettingType) {
	if got := s.Type(); got != want {
		panic(fmt.Sprintf("setting %s: type is %s, not %s", s.s.Key, got, want))
	}
}

// SettingGetByGroup returns all settings belonging to the given group,
// keyed by setting key.
// Settings not present in the database are included with their default values.
func (tx *TxR) SettingGetByGroup(group string) (Settings, error) {
	var err error
	defer errorfmt.Handlef("setting list by group %q: %w", group, &err)
	rows, err := tx.q.SettingListByGroup(group)
	if err != nil {
		return nil, err
	}
	m := make(map[string]*Setting)
	for i := range rows {
		m[rows[i].Key] = &Setting{rows[i]}
	}
	for key, def := range settingDefs {
		if def.Group == group && m[key] == nil {
			m[key] = &Setting{schema.Setting{
				Key:   key,
				Group: group,
				Value: def.defaultJSON(),
			}}
		}
	}
	return m, nil
}

// SettingSetInt sets a setting to the given int value.
func (tx *TxRW) SettingSetInt(key string, n int) error {
	return tx.SettingSet(key, mustMarshalJSON(n))
}

// SettingSetString sets a setting to the given string value.
func (tx *TxRW) SettingSetString(key string, v string) error {
	return tx.SettingSet(key, mustMarshalJSON(v))
}

// SettingSetBool sets a setting to the given bool value.
func (tx *TxRW) SettingSetBool(key string, v bool) error {
	return tx.SettingSet(key, mustMarshalJSON(v))
}

func mustMarshalJSON(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		panic(fmt.Sprintf("marshal setting value: %v", err))
	}
	return string(b)
}

// SettingSet stores a pre-encoded JSON value for the given key,
// validating that the JSON decodes to the correct type for that key.
func (tx *TxRW) SettingSet(key string, rawJSON string) (err error) {
	defer errorfmt.Handlef("setting set %q: %w", key, &err)
	def, ok := settingDefs[key]
	if !ok {
		return fmt.Errorf("unknown setting key %q", key)
	}
	if err := validateSettingJSON(def.Type, rawJSON); err != nil {
		return err
	}
	err = tx.q.SettingSet(schema.SettingSetParams{
		Key:   key,
		Group: def.Group,
		Value: rawJSON,
	})

	if err != nil {
		return err
	}
	s := &Setting{schema.Setting{Key: key, Group: def.Group, Value: rawJSON}}
	for _, f := range settingHooks[key] {
		f(s)
	}
	return nil
}

func validateSettingJSON(typ SettingType, raw string) error {
	switch typ {
	case SettingTypeInt:
		var v int
		return json.Unmarshal([]byte(raw), &v)
	case SettingTypeString:
		var v string
		return json.Unmarshal([]byte(raw), &v)
	case SettingTypeBool:
		var v bool
		return json.Unmarshal([]byte(raw), &v)
	default:
		return fmt.Errorf("unknown setting type %q", typ)
	}
}
