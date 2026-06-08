import { PageHeader } from "./PageHeader";
import { Card } from "@/components/ui/card";
import { Icon } from "./Icon";

export function ComingSoon({
  title,
  description,
  icon,
}: {
  title: string;
  description?: string;
  icon?: string;
}) {
  return (
    <div>
      <PageHeader title={title} description={description} icon={icon} />
      <Card className="card-elev border-border p-10 text-center">
        <div className="w-14 h-14 rounded-2xl bg-brand-500/10 text-brand-600 mx-auto flex items-center justify-center mb-3">
          <Icon name="construction" size={28} />
        </div>
        <h3 className="font-semibold">Coming soon</h3>
        <p className="text-sm text-text-muted mt-1 max-w-md mx-auto">
          This feature is under development. Check back in a future release.
        </p>
      </Card>
    </div>
  );
}
