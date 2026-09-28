import { useEffect, useMemo, useState } from 'react';
import { createRoot } from 'react-dom/client';
import { fixture } from './fixture';
import { FlowDependencies, SimpleDependencies } from './renderers';
import { type DependencyTarget } from '../../contracts/src/index';
import './style.css';
const violations: string[] = [];
document.addEventListener('securitypolicyviolation', (event) =>
  violations.push(event.violatedDirective),
);
Object.assign(window, { qualificationViolations: violations });
function App() {
  const [candidate, setCandidate] = useState('simple'),
    [size, setSize] = useState(8),
    [mounted, setMounted] = useState(true);
  const [selected, select] = useState<DependencyTarget['selection'] | null>(null);
  const observation = useMemo(() => fixture(size), [size]);
  useEffect(() => {
    document.body.dataset.ready = candidate + '-' + size + '-' + mounted;
  }, [candidate, size, mounted]);
  const props = { observation, selected, select };
  return (
    <>
      <header>
        <strong>R3b1 renderer qualification</strong>
        <small>Synthetic fixture · Read-only · No Project or Agent connection</small>
      </header>
      <nav>
        <label>
          Renderer{' '}
          <select
            aria-label="Renderer"
            value={candidate}
            onChange={(e) => {
              select(null);
              setCandidate(e.target.value);
            }}
          >
            <option value="simple">HTML + SVG</option>
            <option value="flow">React Flow + Dagre</option>
          </select>
        </label>
        <label>
          Groups{' '}
          <select
            aria-label="Groups"
            value={size}
            onChange={(e) => {
              select(null);
              setSize(Number(e.target.value));
            }}
          >
            <option>8</option>
            <option>80</option>
            <option>81</option>
          </select>
        </label>
        <button onClick={() => setMounted(!mounted)}>{mounted ? 'Unmount' : 'Mount'}</button>
        <span>
          {observation.groups.length} groups · {observation.edges.length} dependencies
        </span>
      </nav>
      <main>
        <section className="graph-host" aria-label="Dependency graph">
          {!mounted ? null : observation.display !== 'graph' ? (
            <p role="status">Graph limit reached. Use the bounded dependency list.</p>
          ) : candidate === 'simple' ? (
            <SimpleDependencies {...props} />
          ) : (
            <FlowDependencies {...props} />
          )}
        </section>
        <aside>
          <h2>Dependency list</h2>
          {observation.edges.map((edge) => (
            <button
              key={JSON.stringify(edge)}
              onClick={() =>
                select({
                  kind: 'run-dependency',
                  coordinateSpace: 'observed-authored-task-pair',
                  ...edge,
                })
              }
            >
              {edge.fromTaskId} → {edge.toTaskId}
            </button>
          ))}
        </aside>
      </main>
      <footer>
        <strong>Selected target</strong>
        <output id="selection">{selected ? JSON.stringify(selected) : 'None'}</output>
      </footer>
    </>
  );
}
createRoot(document.getElementById('root')!).render(<App />);
