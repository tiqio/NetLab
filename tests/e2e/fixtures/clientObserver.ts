import type { ClientObservation } from "./acceptanceTypes";

export class ClientObserver {
  private readonly observations: ClientObservation[] = [];

  record(observation: ClientObservation) {
    const previous = this.observations.at(-1);
    if (previous && observation.event_sequence <= previous.event_sequence)
      throw new Error("Client event sequence is not ordered");
    this.observations.push(observation);
  }

  assertConverged(
    mutationId: string,
    requiredClients: string[],
    maximumMs = 5000,
  ) {
    const values = this.observations.filter(
      (item) => item.mutation_id === mutationId,
    );
    const revisions = new Set(values.map((item) => item.resource_revision));
    const clients = new Set(values.map((item) => item.client_id));
    if (requiredClients.some((client) => !clients.has(client)))
      throw new Error(
        `Mutation ${mutationId} was not observed by every client`,
      );
    if (revisions.size !== 1)
      throw new Error(
        `Mutation ${mutationId} did not converge to one revision`,
      );
    if (values.some((item) => item.convergence_ms > maximumMs))
      throw new Error(
        `Mutation ${mutationId} exceeded ${maximumMs} ms convergence`,
      );
    return values;
  }

  all() {
    return [...this.observations];
  }
}
