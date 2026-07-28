// RegionSelect — shared AWS region picker used by the OAuth/import flows that
// need a region (BuilderID, IAM SSO, Kiro-SSO, SSO token, Kiro API key).
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { AWS_REGIONS } from '@/config/regions'

interface Props {
  value: string
  onChange: (v: string) => void
  className?: string
}

export function RegionSelect({ value, onChange, className }: Props) {
  return (
    <Select value={value} onValueChange={onChange}>
      <SelectTrigger className={className}>
        <SelectValue />
      </SelectTrigger>
      <SelectContent>
        {AWS_REGIONS.map((r) => (
          <SelectItem key={r} value={r}>
            {r}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  )
}
