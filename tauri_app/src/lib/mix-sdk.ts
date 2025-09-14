import { Mix } from '../../../../mix-typescript-sdk';
import { getBackendUrl } from '@/utils/backendUrl';


// Create a singleton SDK instance configured for our backend
export const mix = new Mix({
  serverURL: getBackendUrl(),
});

export default mix;