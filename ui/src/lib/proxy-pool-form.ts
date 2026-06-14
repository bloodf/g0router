// ProxyPoolForm is the modal's editable form state. port is kept as a string
// (raw <input> value) and coerced to a number in the payload.
export interface ProxyPoolForm {
  name: string;
  protocol: string;
  host: string;
  port: string;
  username: string;
  is_active: boolean;
}

// ProxyPoolCreate is the body sent to POST /api/proxy-pools (new) or
// PUT /api/proxy-pools/{id} (edit). snake_case keys mirror the proxy-pools mock
// shape (§1.2/§1.4); the mock fabricates id/last_check_at itself.
export interface ProxyPoolCreate {
  name: string;
  protocol: string;
  host: string;
  port: number;
  username: string;
  is_active: boolean;
}

// toProxyPoolPayload maps the modal form to the create/edit payload. Pure: it
// coerces the port string to a number (invalid/empty -> 0) and never mutates the
// source form. This is the authoritative proxy-pool-create-contract proof
// (§1.4 point 3).
export function toProxyPoolPayload(form: ProxyPoolForm): ProxyPoolCreate {
  const port = Number.parseInt(form.port, 10);
  return {
    name: form.name,
    protocol: form.protocol,
    host: form.host,
    port: Number.isNaN(port) ? 0 : port,
    username: form.username,
    is_active: form.is_active,
  };
}
