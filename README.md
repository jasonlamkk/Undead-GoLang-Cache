# Undead-GoLang-Cache-Server-Farm


this is a conceptual demo of a undead server-farm

![How it works](https://www.websequencediagrams.com/cgi-bin/cdraw?lz=dGl0bGUgVGhlIFVuZGVhZCBDbHVzdGVyCnBhcnRpY2lwYW50IENsaWVudAAGDU5vZGUgQQABEkIAFBJOCgoAOwYtPgAwBjogTG9uZyBDb21wdXRhdGlvbiBSZXF1ZXN0CgBRBgAfCkNyZWF0ZQAXCCAjVVVJRAAdCC0-PgCBEwY6UmV0dXJuABcHADsJAGgGQjogSSBhbSB3b3JraW5nIG8AIAgAGQ5OAA8YCm5vdGUgb3ZlcgCCAAcsAIF1BwABB046AIIYByB3YWl0IGEgcmVhc29uYWJsZSB0aW1lIGJlZm9yZSBhcwB6BWZvciByZXN1bHQAQRlCAEkJY2FuIGFzayBhbnkgbm9kZSBhYm91dCB0aGUAOwkAgk0NQjogRmV0Y2gAWwcgb2YAgiYMQgCCdgoACxsAglMFAIIyCACCUgcAOBcAgncKABEYAIFIHFdoYXQgaWYAhEMHIG5lZWQgdG8gc2h1dGRvd24_CgCDQQkAgQQNLWJhbGFuYwCBeghzIGV2ZW5seSBiZXR3ZWVuIHBlZXIAgigFczsgZWFjaCBzaGFyZSAxLyhuLTEpIGRhdGEAgVkQTgARRw&s=rose "websequencediagrams")

title The Undead Cluster
participant Client
participant Node A
participant Node B
participant Node N

Client->Node A: Long Computation Request
Node A->Node A: Create Request #UUID
Node A-->>Client:Return #UUID

Node A->>Node B: I am working on #UUID
Node A->>Node N: I am working on #UUID

note over Client, Node A, Node N: Client wait a reasonable time before asking for result

note over Client, Node B: Client can ask any node about the result

Client->Node B: Fetch result of #UUID
Node B->Node A: Fetch result of #UUID
Node A-->>Node B: Return result of #UUID
Node B-->>Client: Return result of #UUID


note over Client, Node B: What if Node A need to shutdown?


Node A-->>Node B: Re-balance results evenly between peer nodes; each share 1/(n-1) data
Node A-->>Node N: Re-balance results evenly between peer nodes; each share 1/(n-1) data
