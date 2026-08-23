import { z } from "zod";
import { exportSchemaJSON } from "@zod2go/export";

const UserSchema = z.object({
  name: z.string().min(2).max(100),
  email: z.email(),
  age: z.number().int().min(0).max(150).optional(),
  role: z.enum(["admin", "user"]).default("user"),
  tags: z.array(z.string()).min(1).optional(),
});

const inputEl = document.querySelector<HTMLTextAreaElement>("#input")!;
const resultEl = document.querySelector<HTMLPreElement>("#result")!;
const irEl = document.querySelector<HTMLPreElement>("#ir")!;
const validateBtn = document.querySelector<HTMLButtonElement>("#validate")!;

inputEl.value = JSON.stringify(
  {
    name: "Grace Hopper",
    email: "grace@example.com",
    tags: ["go", "zod"],
  },
  null,
  2,
);

irEl.textContent = exportSchemaJSON(UserSchema);

function validate(): void {
  let parsed: unknown;
  try {
    parsed = JSON.parse(inputEl.value);
  } catch (err) {
    resultEl.className = "err";
    resultEl.textContent = `Invalid JSON: ${(err as Error).message}`;
    return;
  }

  const result = UserSchema.safeParse(parsed);
  if (result.success) {
    resultEl.className = "ok";
    resultEl.textContent = JSON.stringify(result.data, null, 2);
  } else {
    resultEl.className = "err";
    resultEl.textContent = JSON.stringify(result.error.issues, null, 2);
  }
}

validateBtn.addEventListener("click", validate);
validate();
