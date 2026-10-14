# Commands

Commands that read or modify app data use the configured app and environment
and send the saved token in the `token` request header. Set `app`, `env`, and
`email` with `slingr config KEY VALUE`. Environments are `dev`, `staging`, or
`prod`.

Flag values are parsed as JSON values when valid JSON; otherwise they are
treated as text. Dotted names create nested objects. For example,
`--daySettings.date '2026-10-14'` produces a nested `daySettings` object.

## config

```sh
slingr config
slingr config KEY
slingr config KEY VALUE
```

Lists all configured values, prints one value, or sets a value. Arbitrary keys
are supported. Use `app`, `env`, and `email` to configure a Slingr app.

## login

```sh
slingr login
slingr login PASSWORD
```

Authenticates using the configured email and saves the token for the current
app and environment. If omitted, the password is prompted for without echo.

## query

```http
GET /data/{entityName}?{key}={value}
```

```sh
slingr query holidays --country Argentina
```

## get

```http
GET /data/{entityName}/{recordId}
```

```sh
slingr get holidays d987485b-488e-43c9-b5d4-77ba5dd6950b
```

## new

```http
POST /data/{entityName}
Payload: JSON object built from the supplied flags
```

```sh
slingr new holidays --country Argentina --daySettings.date 2026-10-14
```

This creates a payload equivalent to:

```json
{
  "country": "Argentina",
  "daySettings": {
    "date": "2026-10-14"
  }
}
```

## remove

```http
DELETE /data/{entityName}/{recordId}
```

```sh
slingr remove holidays d987485b-488e-43c9-b5d4-77ba5dd6950b
```

## edit

The CLI fetches the record, merges the supplied fields into it (recursively for
nested objects), and sends the updated record.

```http
GET /data/{entityName}/{recordId}
PUT /data/{entityName}/{recordId}
Payload: merged JSON object
```

```sh
slingr edit holidays d987485b-488e-43c9-b5d4-77ba5dd6950b --daySettings.date 2026-10-10
```

## action

`ACTION_NAME` is required. An optional record ID selects the record action;
without it, the entity-level action is called.

```http
PUT /data/{entityName}/{actionName}
PUT /data/{entityName}/{recordId}/{actionName}
Payload: JSON object built from the supplied flags
```

```sh
slingr action holidays refresh --force true
slingr action holidays approve d987485b-488e-43c9-b5d4-77ba5dd6950b --country USA
```

## logout

```http
POST /auth/logout
```

```sh
slingr logout
```

Logs out and removes the saved token for the configured app/environment.

## token

```sh
slingr token
```

Prints the token saved for the configured app/environment.
