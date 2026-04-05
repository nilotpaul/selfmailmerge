# How to use?

1. Download executable from https://github.com/nilotpaul/selfmailmerge/releases/latest/download/selfmailmerge
2. Create a folder and put the downloaded executable inside.
3. Place your CSV or XLSX file in this folder.
4. Create a `.env` file with your configuration:

should look like this -
```sh
folder/
  selfmailmerge
  leads.csv # or xlsx
  .env
```

env file -
```sh
CSV_FILE_PATH=leads.csv

FROM=mail@cmp.com
SUBJECT=Regarding ..., {{.Name}}
CC=
BCC=
BODY="
Hi, {{.Name}}

This is your email {{.Email}}

Best,
Paul
"
ATTACHMENT=
```

- `CSV_FILE_PATH` → path to your CSV/XLSX file.
- Use `{{.HeaderName}}` to inject values from your spreadsheet headers.
- All spreadsheet data will be accessible from template.

# Run

**Send:**
```sh
selfmailmerge -password=microsoft-account-password
```

**To see how email will look like:**
```sh
selfmailmerge -password=microsoft-account-password -test=true
```
