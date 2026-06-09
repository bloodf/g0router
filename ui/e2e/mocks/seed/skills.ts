import type { Skill } from "../../src/lib/types";

export function seedSkills(): Skill[] {
  return [
    { name: "filesystem", category: "Endpoint Skills", description: "Read and write files", url: "https://github.com/modelcontextprotocol/servers/tree/main/src/filesystem" },
    { name: "github", category: "Endpoint Skills", description: "GitHub API operations", url: "https://github.com/modelcontextprotocol/servers/tree/main/src/github" },
  ];
}
