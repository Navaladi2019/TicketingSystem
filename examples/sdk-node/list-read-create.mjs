import { createClient } from "@nava/sdk";
import { devToken } from "./dev-token.mjs";

const client = createClient({ baseUrl: process.env.NAVA_CORE_URL ?? "http://127.0.0.1:18080", getToken: () => devToken("alice@tenant-a.test") });
let n = 0;
for await (const r of client.records.list("incident", { pageSize: 2 })) console.log(++n, r.number, r.data.short_description);
const made = await client.records.create("incident", { short_description: "Printer on fire (SDK example)" }, { idempotencyKey: process.argv[2] ?? crypto.randomUUID() });
const { record, version } = await client.records.get("incident", made.id);
console.log(`created ${record.number} (${record.id}), version ${version}`);
