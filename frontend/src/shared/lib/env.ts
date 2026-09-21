// The single place in the app allowed to read import.meta.env. Every
// other module imports `env` from here instead of touching
// import.meta.env directly, so a missing/malformed variable fails fast
// at startup with a clear error rather than surfacing as `undefined`
// deep inside a fetch call.

interface AppEnv {
  apiBaseUrl: string;
}

function readEnv(): AppEnv {
  const apiBaseUrl = import.meta.env.VITE_API_BASE_URL;

  // Distinguish "never set" (undefined — genuinely forgotten) from
  // "deliberately empty" (same-origin requests, e.g. behind the nginx
  // reverse proxy in frontend/nginx.conf.template, which forwards /api/
  // to the backend so the browser only ever talks to one origin). A
  // plain falsy check would wrongly reject the second, valid case.
  if (apiBaseUrl === undefined) {
    throw new Error(
      'VITE_API_BASE_URL is not set. Copy .env.example to .env.local and set it.',
    );
  }

  return { apiBaseUrl };
}

export const env: AppEnv = readEnv();
