import type { User } from "../../src/lib/types";

export function seedUsers(): User[] {
  return [
    { id: "user-1", username: "admin", display_name: "Administrator", role: "admin", password: "123456" },
  ];
}
