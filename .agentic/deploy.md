# Deploy Config

## Environments
production: docker build -t g0router . <!-- 3-stage build: node alpine → go alpine → distroless static -->
staging: <!-- optional: add staging deploy command -->

## Version scheme
TODO

## Changelog
path: <!-- no CHANGELOG.md yet -->

## Rollback
command: TODO - fill in once known
notes: <!-- single-binary deploy; rollback = redeploy previous image/binary -->

## Preferences
prefer: production
