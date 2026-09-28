# Workspace preview fixture

quality.png is synthetic test data generated once with Matplotlib. It is not
an analysis result. Its fixed curve is
34 - 0.035*x - 17*(x/150)^6 + 0.65*sin(x/5) for integer x from 1 to 150,
with a ±3 band and an explicit “Illustrative test data” caption.
The committed PNG avoids adding Python or plotting packages to app test/runtime
dependencies. Electron tests copy it into an isolated temporary Project and
read it through the actual Go service and validated workspace bridge.
