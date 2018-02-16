# Changelog

## v1.2.0 (unreleased)

- Add a `Reserved(quota float64)` option to allow users to reserve part of
  their CPU quota things outside the Go runtime.

## v1.1.0 (2017-11-10)

- Log the new value of `GOMAXPROCS` rather than the current value.
- Make logs more explicit about whether `GOMAXPROCS` was modified or not.
- Allow customization of the minimum `GOMAXPROCS`, and modify default from 2 to 1.

## v1.0.0 (2017-08-09)

- Initial release.
