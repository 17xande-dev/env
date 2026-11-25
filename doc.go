// Package env reads environment settings into typed values and collects
// configuration problems without echoing the input. Read the settings, report
// any additional checks with [Reader.Fail], then check [Reader.Err] before
// using the configuration.
package env
