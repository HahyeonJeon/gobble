import { Component, type ErrorInfo, type ReactNode } from 'react';

export class ErrorBoundary extends Component<{ children: ReactNode }, { failed: boolean }> {
  state = { failed: false };
  static getDerivedStateFromError() {
    return { failed: true };
  }
  componentDidCatch(_error: Error, _info: ErrorInfo): void {
    /* No Project content is logged. */
  }
  render() {
    if (this.state.failed)
      return (
        <div className="empty-state" role="alert">
          <h2>This view could not be displayed</h2>
          <p>Your saved workspace is still available.</p>
          <button onClick={() => this.setState({ failed: false })}>Try again</button>
        </div>
      );
    return this.props.children;
  }
}
