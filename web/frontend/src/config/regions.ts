// AWS regions offered in the OAuth/import flows that need one. The old app used
// this same short list; us-east-1 is the default for Kiro/BuilderID.
export const AWS_REGIONS = [
  'us-east-1',
  'us-west-2',
  'eu-central-1',
  'eu-west-1',
  'ap-southeast-1',
  'ap-southeast-2',
  'ap-northeast-1',
] as const

export const DEFAULT_REGION = 'us-east-1'
