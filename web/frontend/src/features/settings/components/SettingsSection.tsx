// SettingsSection — a titled card wrapper with an id anchor for the scroll-spy
// nav. Each settings domain renders one; the nav observes their ids.
import type { ReactNode } from 'react'
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card'

interface Props {
  id: string
  title: string
  description?: string
  children: ReactNode
}

export function SettingsSection({ id, title, description, children }: Props) {
  return (
    <Card id={id} className="scroll-mt-20">
      <CardHeader>
        <CardTitle>{title}</CardTitle>
        {description && <CardDescription>{description}</CardDescription>}
      </CardHeader>
      <CardContent className="space-y-4">{children}</CardContent>
    </Card>
  )
}
