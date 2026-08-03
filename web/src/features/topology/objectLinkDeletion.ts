import type {
  NetworkObjectLink,
  NetworkObjectLinkTaskEnvelope,
  OperationTask,
} from "@/api";

export interface ObjectLinkDeletionDependencies {
  hide: (id: string) => void;
  clearSelection: () => void;
  submit: (link: NetworkObjectLink) => Promise<NetworkObjectLinkTaskEnvelope>;
  recordTask: (task: OperationTask) => void;
  unhide: (id: string) => void;
  reload: () => Promise<void>;
}

export async function runObjectLinkDeletion(
  link: NetworkObjectLink,
  dependencies: ObjectLinkDeletionDependencies,
) {
  dependencies.hide(link.id);
  dependencies.clearSelection();
  try {
    const envelope = await dependencies.submit(link);
    dependencies.recordTask(envelope.task);
    return envelope;
  } catch (error) {
    dependencies.unhide(link.id);
    await dependencies.reload();
    throw error;
  }
}
