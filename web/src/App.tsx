import { useEffect, useState } from 'react';
import { getApplicationInfo } from './api/generated/sdk.gen';

export function App() {
  const [name, setName] = useState('');
  const [error, setError] = useState(false);

  useEffect(() => {
    const controller = new AbortController();
    void getApplicationInfo({ signal: controller.signal }).then((response) => {
      if (controller.signal.aborted) return;
      if (response.data) setName(response.data.name);
      else setError(true);
    }).catch(() => {
      if (!controller.signal.aborted) setError(true);
    });
    return () => controller.abort();
  }, []);

  return (
    <main>
      <h1>Hello World</h1>
      <p>{name || 'Loading application…'}</p>
      {error && <p role="alert">Application information is unavailable. Reload to retry.</p>}
    </main>
  );
}
