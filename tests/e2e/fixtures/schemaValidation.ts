import Ajv2020, {
  type ErrorObject,
  type ValidateFunction,
} from "ajv/dist/2020.js";
import addFormats from "ajv-formats";
import { readFile } from "node:fs/promises";
import { basename, resolve } from "node:path";
import type {
  AcceptanceEvidence,
  InteractionDefinition,
} from "./acceptanceTypes";

const repositoryRoot =
  basename(process.cwd()) === "web"
    ? resolve(process.cwd(), "..")
    : process.cwd();
const contractsDirectory = resolve(
  repositoryRoot,
  "specs/003-frontend-interaction-acceptance/contracts",
);

async function compile(name: string): Promise<ValidateFunction> {
  const schema = JSON.parse(
    await readFile(`${contractsDirectory}/${name}`, "utf8"),
  ) as object;
  const ajv = new Ajv2020({ allErrors: true, strict: true });
  addFormats(ajv);
  return ajv.compile(schema);
}

function describeErrors(errors: ErrorObject[] | null | undefined) {
  return (errors || [])
    .map((error) => `${error.instancePath || "/"} ${error.message}`)
    .join("; ");
}

export async function validateInventory(
  value: unknown,
): Promise<InteractionDefinition[]> {
  const validate = await compile("interaction-inventory.schema.json");
  if (!validate(value)) {
    throw new Error(
      `Invalid interaction inventory: ${describeErrors(validate.errors)}`,
    );
  }
  const interactions = (value as { interactions: InteractionDefinition[] })
    .interactions;
  const seen = new Set<string>();
  for (const interaction of interactions) {
    if (seen.has(interaction.id)) {
      throw new Error(`Duplicate interaction id: ${interaction.id}`);
    }
    seen.add(interaction.id);
  }
  return interactions;
}

export async function validateEvidence(
  value: unknown,
): Promise<AcceptanceEvidence> {
  const validate = await compile("acceptance-evidence.schema.json");
  if (!validate(value)) {
    throw new Error(
      `Invalid acceptance evidence: ${describeErrors(validate.errors)}`,
    );
  }
  return value as AcceptanceEvidence;
}
