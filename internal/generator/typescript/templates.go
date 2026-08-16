package typescript

const expandHelperSource = `const envVarRefRx = /\$\{[A-Za-z_][A-Za-z0-9_]*\}/g;

function expandVars(s: string, m: Record<string, string | undefined>): string {
  return s.replace(envVarRefRx, (ref) => {
    const v = m[ref.slice(2, -1)];

    if (v !== undefined && v !== "") {
      return v;
    }

    return ref;
  });
}
`

const parseIntStrictSource = `function parseIntStrict(value: string): number | null {
  if (!/^[+-]?\d+$/.test(value)) {
    return null;
  }

  return Number.parseInt(value, 10);
}
`

const parseBoolStrictSource = `function parseBoolStrict(value: string): boolean | null {
  if (value === "1" || value === "t" || value === "T" ||
    value === "true" || value === "TRUE" || value === "True") {
    return true;
  }

  if (value === "0" || value === "f" || value === "F" ||
    value === "false" || value === "FALSE" || value === "False") {
    return false;
  }

  return null;
}
`

const parseFloatStrictSource = `function parseFloatStrict(value: string): number | null {
  if (/^[+-]?nan$/i.test(value)) {
    return Number.NaN;
  }

  if (/^[+-]?(inf|infinity)$/i.test(value)) {
    return value.startsWith("-") ? Number.NEGATIVE_INFINITY : Number.POSITIVE_INFINITY;
  }

  if (!/^[+-]?(\d+(\.\d*)?|\.\d+)([eE][+-]?\d+)?$/.test(value)) {
    return null;
  }

  return Number(value);
}
`

const isValidURLSource = `function isValidUrl(value: string): boolean {
  try {
    new URL(value);
    return true;
  } catch {
    return false;
  }
}
`
