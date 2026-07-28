// Shared props for every add-account flow component. onDone closes the wizard
// after a successful add; the flow itself invalidates the accounts query.
export interface FlowComponentProps {
  onDone: () => void
}
