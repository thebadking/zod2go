# 03 — Demo playground

Browser playground for validating JSON against Zod and previewing exported IR.

## Deliverables

| Item | Path |
| ---- | ---- |
| HTML shell | `index.html` |
| Demo script | `demo/main.ts` |
| Vite config | `ts/vite.config.ts` — root `..`, alias `@zod2go/export`, host `0.0.0.0:4020`, `allowedHosts` |
| Dev command | `npm run dev -- --host 0.0.0.0 --port 4020` |

## Behavior

- Textarea for JSON input with sample user payload.
- **Validate with Zod** runs `safeParse` and shows issues or parsed output.
- **Exported IR** panel shows `exportSchemaJSON(UserSchema)`.

## Acceptance

Dev server loads at configured port; validate button works without console errors.
