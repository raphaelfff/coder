import { makeStyles } from "@mui/styles";
import { useMachine } from "@xstate/react";
import { Alert } from "components/Alert/Alert";
import { ConfirmDialog } from "components/Dialogs/ConfirmDialog/ConfirmDialog";
import { Loader } from "components/Loader/Loader";
import { PageHeader, PageHeaderTitle } from "components/PageHeader/PageHeader";
import dayjs from "dayjs";
import {
  scheduleToAutostart,
  scheduleChanged,
} from "pages/WorkspaceSettingsPage/WorkspaceSchedulePage/schedule";
import { ttlMsToAutostop } from "pages/WorkspaceSettingsPage/WorkspaceSchedulePage/ttl";
import { useWorkspaceSettings } from "pages/WorkspaceSettingsPage/WorkspaceSettingsLayout";
import { FC } from "react";
import { Helmet } from "react-helmet-async";
import { Navigate, useNavigate, useParams } from "react-router-dom";
import { pageTitle } from "utils/page";
import * as TypesGen from "api/typesGenerated";
import { workspaceByOwnerAndNameKey } from "api/queries/workspace";
import { WorkspaceScheduleForm } from "./WorkspaceScheduleForm";
import { workspaceSchedule } from "xServices/workspaceSchedule/workspaceScheduleXService";
import {
  formValuesToAutostartRequest,
  formValuesToTTLRequest,
} from "./formToRequest";
import { ErrorAlert } from "components/Alert/ErrorAlert";
import { useQueryClient } from "@tanstack/react-query";

const getAutostart = (workspace: TypesGen.Workspace) =>
  scheduleToAutostart(workspace.autostart_schedule);
const getAutostop = (workspace: TypesGen.Workspace) =>
  ttlMsToAutostop(workspace.ttl_ms);

const useStyles = makeStyles((theme) => ({
  topMargin: {
    marginTop: theme.spacing(3),
  },
  pageHeader: {
    paddingTop: 0,
  },
}));

export const WorkspaceSchedulePage: FC = () => {
  const styles = useStyles();
  const params = useParams() as { username: string; workspace: string };
  const navigate = useNavigate();
  const username = params.username.replace("@", "");
  const workspaceName = params.workspace;
  const queryClient = useQueryClient();
  const workspace = useWorkspaceSettings();
  const [scheduleState, scheduleSend] = useMachine(workspaceSchedule, {
    context: { workspace },
  });
  const {
    checkPermissionsError,
    submitScheduleError,
    getTemplateError,
    permissions,
    template,
  } = scheduleState.context;

  if (!username || !workspaceName) {
    return <Navigate to="/workspaces" />;
  }

  if (scheduleState.matches("done")) {
    return <Navigate to={`/@${username}/${workspaceName}`} />;
  }

  return (
    <>
      <Helmet>
        <title>{pageTitle([workspaceName, "Schedule"])}</title>
      </Helmet>
      <PageHeader className={styles.pageHeader}>
        <PageHeaderTitle>Workspace Schedule</PageHeaderTitle>
      </PageHeader>
      {(scheduleState.hasTag("loading") || !template) && <Loader />}
      {scheduleState.matches("error") && (
        <ErrorAlert error={checkPermissionsError || getTemplateError} />
      )}
      {permissions && !permissions.updateWorkspace && (
        <Alert severity="error">
          You don&apos;t have permissions to update the schedule for this
          workspace.
        </Alert>
      )}
      {template &&
        workspace &&
        (scheduleState.matches("presentForm") ||
          scheduleState.matches("submittingSchedule")) && (
          <WorkspaceScheduleForm
            submitScheduleError={submitScheduleError}
            initialValues={{
              ...getAutostart(workspace),
              ...getAutostop(workspace),
            }}
            isLoading={scheduleState.tags.has("loading")}
            defaultTTL={dayjs.duration(template.default_ttl_ms, "ms").asHours()}
            onCancel={() => {
              navigate(`/@${username}/${workspaceName}`);
            }}
            onSubmit={async (values) => {
              scheduleSend({
                type: "SUBMIT_SCHEDULE",
                autostart: formValuesToAutostartRequest(values),
                ttl: formValuesToTTLRequest(values),
                autostartChanged: scheduleChanged(
                  getAutostart(workspace),
                  values,
                ),
                autostopChanged: scheduleChanged(
                  getAutostop(workspace),
                  values,
                ),
              });

              await queryClient.invalidateQueries(
                workspaceByOwnerAndNameKey(params.username, params.workspace),
              );
            }}
          />
        )}
      <ConfirmDialog
        open={scheduleState.matches("showingRestartDialog")}
        title="Restart workspace?"
        description="Would you like to restart your workspace now to apply your new autostop setting, or let it apply after your next workspace start?"
        confirmText="Restart"
        cancelText="Apply later"
        hideCancel={false}
        onConfirm={() => {
          scheduleSend("RESTART_WORKSPACE");
        }}
        onClose={() => {
          scheduleSend("APPLY_LATER");
        }}
      />
    </>
  );
};

export default WorkspaceSchedulePage;
