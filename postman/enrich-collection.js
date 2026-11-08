#!/usr/bin/env node
// Re-applies the hand-authored parts to the generated Postman collection.
//
// The collection is a build artifact of `openapi2postmanv2` (see the command in
// docs/FEAT-003/dev-test-results.md) — regenerating it overwrites anything hand
// added, so the additions live here instead of in the JSON. Run after every
// regeneration:
//
//   npx openapi-to-postmanv2 -s api/openapi.yaml -o postman/collection.json -p -O folderStrategy=Tags
//   node postman/enrich-collection.js
//
// Four things are added:
//
//   1. The collection-level tests. Structural only — a JSON body *when the route
//      answers JSON* and a sane latency. The export route answers text/csv, so
//      the body test has to look at the content type or it fails on a pass.
//   2. FEAT-003's export requests. The generator does emit the route from
//      api/openapi.yaml, but a generated request inherits the collection's one
//      token, so it cannot also cover "another user sees only their own rows",
//      "anonymous" and "invalid token" — those need per-request auth overrides,
//      which regeneration wipes.
//   3. FEAT-005's summary requests, for the same reason: the seeded totals are
//      per-user, so the multi-user, ignored-input and 401 cases need their own
//      tokens/overrides. The named FE surface gets its own folder because the
//      badge tooltip is a screen, not an API tag.
//   4. The metrics scrape, which is not an API route at all.
//
// Per-route expectations are asserted in the scripts below and in Newman's own
// exit code, not baked into the collection-level test as business values.

const fs = require('fs')
const path = require('path')

const file = path.join(__dirname, 'collection.json')
const collection = JSON.parse(fs.readFileSync(file, 'utf8'))

// Marks the injected requests so a second run replaces rather than duplicates
// them. Regeneration wipes them anyway; this only matters if someone runs the
// script twice by mistake.
const INJECTED = '__feat003'
const INJECTED_SUMMARY = '__feat005'
const INJECTED_MARKERS = [INJECTED, INJECTED_SUMMARY]

const script = (lines) => ({
  listen: 'test',
  script: { type: 'text/javascript', exec: lines.join('\n').split('\n') },
})

collection.event = [
  script([
    "pm.test('responds with a JSON body when the route is JSON', () => {",
    "  const type = pm.response.headers.get('Content-Type') || ''",
    "  if (type.includes('application/json')) {",
    '    pm.response.json()',
    '  }',
    '})',
    "pm.test('responds within 2s', () => {",
    '  pm.expect(pm.response.responseTime).to.be.below(2000)',
    '})',
  ]),
]

const bearer = (value) => ({
  type: 'apikey',
  apikey: [
    { key: 'key', value: 'Authorization', type: 'string' },
    { key: 'value', value, type: 'string' },
    { key: 'in', value: 'header', type: 'string' },
  ],
})

const exportURL = (query) => {
  const pairs = query || []
  const suffix = pairs.length
    ? '?' + pairs.map((pair) => `${pair.key}=${encodeURIComponent(pair.value)}`).join('&')
    : ''

  return {
    raw: '{{baseUrl}}/v1/favorites/export' + suffix,
    host: ['{{baseUrl}}'],
    path: ['v1', 'favorites', 'export'],
    query: pairs,
    variable: [],
  }
}

// The seeded fixture (tests/live/postgres-seed.sql). Kept as one literal so a
// mismatch between the fixture and the renderer fails here, on the wire, rather
// than being papered over by a per-line regex.
//
// The first request carries ?page=2&category=Office purely to prove the route
// binds neither: the body below is the complete list, unfiltered and unpaged.
const HEADER = 'item_id,name,category,price,available,favorited_at,note\r\n'
const USER1_CSV =
  HEADER +
  '103,Pen,,0,Tersedia,2026-04-11,Café — 🎁\r\n' +
  "102,'=cmd|calc,Office,1500,Tidak Tersedia,2026-04-10,'=1+1\r\n" +
  '101,"Notebook, A5",Office,19.99,Tersedia,2026-04-09,Buy more\r\n' +
  '104,"Line1\nLine2",Misc,1234.5678,Tersedia,2026-04-01,\r\n' +
  '105,"He said ""hi"", twice",,7.5,Tersedia,2026-03-05,\r\n'

const exportRequests = [
  {
    name: `${INJECTED} Export favorites as CSV`,
    request: {
      method: 'GET',
      header: [{ key: 'Accept', value: 'text/csv' }],
      url: exportURL([
        { key: 'page', value: '2', disabled: false },
        { key: 'category', value: 'Office', disabled: false },
      ]),
      auth: bearer('{{apiKeyUser1}}'),
      description:
        'User 1 has five favorites, seeded to exercise quoting, an embedded ' +
        'newline, a comma, doubled quotes, a non-ASCII note, a discontinued ' +
        'item, an integer price and formula neutralisation. Public request ' +
        'inputs that are named in the cases but not supported by the route ' +
        '(`page`, `category`) must change nothing. The body is asserted byte ' +
        'for byte apart from the BOM, which is asserted as the first three bytes.',
    },
    event: [
      script([
        "pm.test('status 200', () => pm.response.to.have.status(200))",
        "pm.test('Content-Type is bare text/csv, no charset parameter', () => {",
        "  pm.expect(pm.response.headers.get('Content-Type')).to.eql('text/csv')",
        '})',
        "pm.test('Content-Disposition is a quoted attachment filename', () => {",
        "  const value = pm.response.headers.get('Content-Disposition') || ''",
        '  pm.expect(value).to.match(/^attachment; filename="favorites-\\d+-\\d{4}-\\d{2}-\\d{2}\\.csv"$/)',
        '})',
        "pm.test('Cache-Control is no-store', () => {",
        "  pm.expect(pm.response.headers.get('Cache-Control')).to.eql('no-store')",
        '})',
        "pm.test('file starts with the UTF-8 BOM', () => {",
        '  const hex = Buffer.from(pm.response.stream).toString(\'hex\').slice(0, 6)',
        "  pm.expect(hex).to.eql('efbbbf')",
        '})',
        "pm.test('body is the header plus every row, newest first', () => {",
        '  const expected = ' + JSON.stringify(USER1_CSV),
        "  const body = pm.response.text().replace(/^\\uFEFF/, '')",
        '  pm.expect(body).to.eql(expected)',
        '})',
      ]),
    ],
  },
  {
    name: `${INJECTED} Export favorites — caller-supplied user id is ignored`,
    request: {
      method: 'GET',
      header: [
        { key: 'Accept', value: 'text/csv' },
        { key: 'X-User-Id', value: '1' },
      ],
      url: exportURL([{ key: 'user_id', value: '1', disabled: false }]),
      auth: bearer('{{apiKeyUser3}}'),
      description:
        'User 3 has no favorites. `?user_id=1` and `X-User-Id: 1` are both ' +
        'unbound, so the file must be the header row alone — user 1\'s five ' +
        'rows must not appear.',
    },
    event: [
      script([
        "pm.test('status 200', () => pm.response.to.have.status(200))",
        "pm.test('header-only file', () => {",
        "  const body = pm.response.text().replace(/^\\uFEFF/, '')",
        '  pm.expect(body).to.eql(' + JSON.stringify(HEADER) + ')',
        '})',
      ]),
    ],
  },
  {
    name: `${INJECTED} Export favorites — anonymous`,
    request: {
      method: 'GET',
      header: [{ key: 'Accept', value: 'text/csv' }],
      url: exportURL([{ key: 'user_id', value: '1', disabled: false }]),
      auth: { type: 'noauth' },
    },
    event: [
      script([
        "pm.test('status 401', () => pm.response.to.have.status(401))",
        "pm.test('shared error envelope, not a file', () => {",
        "  pm.expect(pm.response.headers.get('Content-Disposition')).to.be.oneOf([null, undefined])",
        "  pm.expect(pm.response.json().payload.code).to.eql('E-1-CMD-AUT-001')",
        '})',
      ]),
    ],
  },
  {
    name: `${INJECTED} Export favorites — invalid token`,
    request: {
      method: 'GET',
      header: [{ key: 'Accept', value: 'text/csv' }],
      url: exportURL(),
      auth: bearer('{{invalidToken}}'),
    },
    event: [
      script([
        "pm.test('status 401', () => pm.response.to.have.status(401))",
        "pm.test('same envelope code as anonymous', () => {",
        "  pm.expect(pm.response.json().payload.code).to.eql('E-1-CMD-AUT-001')",
        '})',
      ]),
    ],
  },
]

const summaryURL = (query) => {
  const pairs = query || []
  const suffix = pairs.length
    ? '?' + pairs.map((pair) => `${pair.key}=${encodeURIComponent(pair.value)}`).join('&')
    : ''

  return {
    raw: '{{baseUrl}}/v1/favorites/summary' + suffix,
    host: ['{{baseUrl}}'],
    path: ['v1', 'favorites', 'summary'],
    query: pairs,
    variable: [],
  }
}

// Body-shape assertion shared by the summary requests: the payload is exactly
// the two integers, and nothing else leaked into it.
const summaryAssertions = (total, discontinued) => [
  "pm.test('status 200', () => pm.response.to.have.status(200))",
  "pm.test('success envelope with exactly the expected integers', () => {",
  '  const body = pm.response.json()',
  '  pm.expect(body.success).to.be.true',
  `  pm.expect(body.payload).to.eql({ total: ${total}, discontinued: ${discontinued} })`,
  '})',
  "pm.test('Content-Type is application/json', () => {",
  "  pm.expect(pm.response.headers.get('Content-Type')).to.include('application/json')",
  '})',
]

// tests/live/postgres-seed.sql: user 1 owns items 101-105 and item 102 is
// discontinued, so {5,1}; user 3 owns nothing, so {0,0}.
const summaryRequests = [
  {
    name: `${INJECTED_SUMMARY} Summary — user 1 (5 total, 1 discontinued)`,
    request: {
      method: 'GET',
      header: [],
      url: summaryURL(),
      auth: bearer('{{apiKeyUser1}}'),
      description:
        'Seed: user 1 owns five favorites and item 102 is discontinued. Both ' +
        'counts are JSON integers; total is not availability-filtered.',
    },
    event: [script(summaryAssertions(5, 1))],
  },
  {
    name: `${INJECTED_SUMMARY} Summary — user 3 (zero favorites)`,
    request: {
      method: 'GET',
      header: [],
      url: summaryURL(),
      auth: bearer('{{apiKeyUser3}}'),
      description: 'Zero favorites is a 200 with both values 0, never 404 or an empty body.',
    },
    event: [script(summaryAssertions(0, 0))],
  },
  {
    name: `${INJECTED_SUMMARY} Summary — caller-supplied user id and counts are ignored`,
    request: {
      method: 'GET',
      header: [{ key: 'X-User-Id', value: '1' }],
      url: summaryURL([
        { key: 'user_id', value: '1', disabled: false },
        { key: 'total', value: '999', disabled: false },
        { key: 'discontinued', value: '999', disabled: false },
      ]),
      auth: bearer('{{apiKeyUser3}}'),
      description:
        'User 3 has no favorites. Asking for user 1\'s data and supplying ' +
        'counts must change nothing — user 1\'s {5,1} must not appear.',
    },
    event: [script(summaryAssertions(0, 0))],
  },
  {
    name: `${INJECTED_SUMMARY} Summary — anonymous`,
    request: {
      method: 'GET',
      header: [],
      url: summaryURL(),
      auth: { type: 'noauth' },
    },
    event: [
      script([
        "pm.test('status 401', () => pm.response.to.have.status(401))",
        "pm.test('shared error envelope, no summary payload', () => {",
        "  const body = pm.response.json()",
        '  pm.expect(body.success).to.be.false',
        "  pm.expect(body.payload.code).to.eql('E-1-CMD-AUT-001')",
        '  pm.expect(body.payload.total).to.be.undefined',
        '  pm.expect(body.payload.discontinued).to.be.undefined',
        '})',
      ]),
    ],
  },
  {
    name: `${INJECTED_SUMMARY} Summary — invalid token`,
    request: {
      method: 'GET',
      header: [],
      url: summaryURL(),
      auth: bearer('{{invalidToken}}'),
    },
    event: [
      script([
        "pm.test('status 401', () => pm.response.to.have.status(401))",
        "pm.test('same envelope code as anonymous', () => {",
        "  pm.expect(pm.response.json().payload.code).to.eql('E-1-CMD-AUT-001')",
        '})',
      ]),
    ],
  },
]

// The named FE surface: the FEAT-004 badge's hover tooltip. One folder per
// screen, holding the request whose response the tooltip renders.
const FE_FOLDER = 'AppShell badge tooltip'
const feRequests = [
  {
    name: `${INJECTED_SUMMARY} AppShell badge tooltip — summary shown on hover`,
    request: {
      method: 'GET',
      header: [],
      url: summaryURL(),
      auth: bearer('{{apiKeyUser1}}'),
      description:
        'What the tooltip renders: `N favorites (M discontinued)` from the ' +
        'response integers. The badge itself shows N and is out of scope here.',
    },
    event: [script(summaryAssertions(5, 1))],
  },
]

const metricsRequests = [
  {
    name: `${INJECTED} Scrape metrics`,
    request: {
      method: 'GET',
      header: [],
      url: {
        raw: '{{baseUrl}}/metrics',
        host: ['{{baseUrl}}'],
        path: ['metrics'],
        query: [],
        variable: [],
      },
      auth: { type: 'noauth' },
    },
    event: [
      script([
        "pm.test('status 200', () => pm.response.to.have.status(200))",
        "const body = pm.response.text()",
        "pm.test('the export route is observed under its own path template', () => {",
        '  pm.expect(body).to.match(/my_app_http_hist_count\\{method="GET",path="\\/v1\\/favorites\\/export",status="200"\\} \\d+/)',
        '})',
        "pm.test('a rejected export is observed as 401, not 200', () => {",
        '  pm.expect(body).to.match(/my_app_http_hist_count\\{method="GET",path="\\/v1\\/favorites\\/export",status="401"\\} [1-9]\\d*/)',
        '})',
        "pm.test('the summary route is observed under its own path template', () => {",
        '  pm.expect(body).to.match(/my_app_http_hist_count\\{method="GET",path="\\/v1\\/favorites\\/summary",status="200"\\} \\d+/)',
        '})',
        "pm.test('a rejected summary is observed as 401, not 200', () => {",
        '  pm.expect(body).to.match(/my_app_http_hist_count\\{method="GET",path="\\/v1\\/favorites\\/summary",status="401"\\} [1-9]\\d*/)',
        '})',
        "pm.test('exactly one summary 200 sample — no duplicate labels', () => {",
        '  const matches = body.match(/^my_app_http_hist_count\\{method="GET",path="\\/v1\\/favorites\\/summary",status="200"\\} \\d+$/gm) || []',
        '  pm.expect(matches.length).to.eql(1)',
        '})',
        "pm.test('the summary route is not relabelled with a query string', () => {",
        '  pm.expect(body).to.not.match(/path="\\/v1\\/favorites\\/summary\\?/)',
        '})',
        "pm.test('/metrics does not count itself', () => {",
        '  pm.expect(body).to.not.match(/path="\\/metrics"/)',
        '})',
      ]),
    ],
  },
]

// Remove a previous injection so the script is idempotent, and drop the
// generator's own export request: it is the same route with the collection's
// one token and no assertions, which the request injected below supersedes.
//
// It is the only generated request removed. The other three are left exactly as
// the generator produced them.
const generatedExport = (request) =>
  request.request?.method === 'GET' &&
  (request.request?.url?.path || []).join('/') === 'v1/favorites/export'

const generatedSummary = (request) =>
  request.request?.method === 'GET' &&
  (request.request?.url?.path || []).join('/') === 'v1/favorites/summary'

collection.item = (collection.item || []).filter(
  (folder) => folder.name !== 'Observability' && folder.name !== FE_FOLDER
)
for (const folder of collection.item) {
  folder.item = (folder.item || []).filter(
    (request) =>
      !INJECTED_MARKERS.some((marker) => String(request.name).startsWith(marker)) &&
      !generatedExport(request) &&
      !generatedSummary(request)
  )
}

const favoriteFolder = (collection.item || []).find((folder) => folder.name === 'Favorite')
if (!favoriteFolder) {
  throw new Error('no "Favorite" folder in the generated collection — did folderStrategy=Tags change?')
}

const after = favoriteFolder.item.findIndex((request) => request.name === "List the calling user's favorites")
if (after === -1) {
  throw new Error('could not find the favorites list request to anchor the export requests to')
}
favoriteFolder.item.splice(after + 1, 0, ...summaryRequests, ...exportRequests)

collection.item.push({
  name: FE_FOLDER,
  description:
    'The FEAT-004 badge on the AppShell: hovering it shows the FEAT-005 summary as `N favorites (M discontinued)`.',
  item: feRequests,
})

collection.item.push({
  name: 'Observability',
  description:
    'The Prometheus scrape. Not an API route; used to verify the export and summary routes are measured.',
  item: metricsRequests,
})

// The converter fills a parameter's value with a type hint (`<long>` here)
// rather than a usable value — in a path variable, and in this API's one
// numeric JSON body field. Neither is something Newman can send, so both are
// pointed at the environment variable the run is otherwise driven by.
//
// The body placeholder is written *without* surrounding quotes so the
// substituted value stays a JSON number rather than becoming a string.
const PLACEHOLDERS = ['<long>', '<integer>', '<int32>', '<int64>', '<number>']

function detype(text) {
  let out = text
  for (const placeholder of PLACEHOLDERS) {
    out = out.split(`"${placeholder}"`).join('{{item_id}}')
    out = out.split(placeholder).join('{{item_id}}')
  }
  return out
}

let rewritten = 0
for (const folder of collection.item || []) {
  for (const request of folder.item || []) {
    for (const variable of request.request?.url?.variable || []) {
      const next = detype(String(variable.value))
      if (next !== variable.value) {
        variable.value = next
        rewritten++
      }
    }

    const body = request.request?.body
    if (body && typeof body.raw === 'string') {
      const next = detype(body.raw)
      if (next !== body.raw) {
        body.raw = next
        rewritten++
      }
    }
  }
}

fs.writeFileSync(file, JSON.stringify(collection, null, 2) + '\n')
console.log(
  `added collection-level tests, ${summaryRequests.length} summary requests, ` +
    `${feRequests.length} FE-screen request, ${exportRequests.length} export requests and the metrics scrape; ` +
    `rewrote ${rewritten} path-parameter placeholder(s)`
)
