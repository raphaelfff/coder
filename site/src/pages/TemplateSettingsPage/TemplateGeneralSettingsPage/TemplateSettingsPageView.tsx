import { Template, UpdateTemplateMeta } from "api/typesGenerated";
import { ComponentProps, FC } from "react";
import { TemplateSettingsForm } from "./TemplateSettingsForm";
import { PageHeader, PageHeaderTitle } from "components/PageHeader/PageHeader";
import { makeStyles } from "@mui/styles";

export interface TemplateSettingsPageViewProps {
  template: Template;
  onSubmit: (data: UpdateTemplateMeta) => void;
  onCancel: () => void;
  isSubmitting: boolean;
  submitError?: unknown;
  initialTouched?: ComponentProps<
    typeof TemplateSettingsForm
  >["initialTouched"];
}

export const TemplateSettingsPageView: FC<TemplateSettingsPageViewProps> = ({
  template,
  onCancel,
  onSubmit,
  isSubmitting,
  submitError,
  initialTouched,
}) => {
  const styles = useStyles();

  return (
    <>
      <PageHeader className={styles.pageHeader}>
        <PageHeaderTitle>General Settings</PageHeaderTitle>
      </PageHeader>

      <TemplateSettingsForm
        initialTouched={initialTouched}
        isSubmitting={isSubmitting}
        template={template}
        onSubmit={onSubmit}
        onCancel={onCancel}
        error={submitError}
      />
    </>
  );
};

const useStyles = makeStyles(() => ({
  pageHeader: {
    paddingTop: 0,
  },
}));
