import 'package:flutter/material.dart';
import 'package:intl/intl.dart';

import 'package:antinvestor_api_runtime/antinvestor_api_runtime.dart' as rt;

import 'json_block.dart';
import 'status_helpers.dart';
import 'trustage_status_badge.dart';

/// Converts a protobuf Struct to a Dart Map.
Map<String, dynamic> _structToMap(dynamic s) {
  if (s == null) return {};
  try {
    final fields = (s as dynamic).fields as Map;
    return fields.map((k, v) => MapEntry(k.toString(), _valueToNative(v)));
  } catch (_) {
    return {};
  }
}

dynamic _valueToNative(dynamic v) {
  if (v == null) return null;
  try {
    if (v.hasStringValue()) return v.stringValue;
    if (v.hasNumberValue()) return v.numberValue;
    if (v.hasBoolValue()) return v.boolValue;
    if (v.hasStructValue()) return _structToMap(v.structValue);
    if (v.hasListValue()) {
      return (v.listValue.values as List).map(_valueToNative).toList();
    }
  } catch (_) {}
  return v.toString();
}

/// Visual execution timeline showing all attempts, scope runs, signals,
/// and child instances for a workflow run.
class ExecutionGraph extends StatelessWidget {
  const ExecutionGraph({
    super.key,
    required this.run,
    this.selectedExecutionId,
    this.onSelectExecution,
    this.onSelectInstance,
  });

  final rt.GetInstanceRunResponse run;
  final String? selectedExecutionId;
  final ValueChanged<String>? onSelectExecution;
  final ValueChanged<String>? onSelectInstance;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final inst = run.instance;
    final executions = List<rt.WorkflowExecution>.from(run.executions)
      ..sort((a, b) => a.createdAt.seconds.compareTo(b.createdAt.seconds));

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        // Instance header
        _InstanceHeader(instance: inst, run: run),
        const SizedBox(height: 16),

        // Execution timeline
        if (executions.isEmpty)
          Center(
            child: Padding(
              padding: const EdgeInsets.all(24),
              child: Text(
                'No executions recorded yet.',
                style: theme.textTheme.bodySmall?.copyWith(
                  color: theme.colorScheme.onSurfaceVariant,
                ),
              ),
            ),
          )
        else
          ...executions.map((exec) => _ExecutionNode(
                execution: exec,
                isSelected: exec.id == selectedExecutionId,
                onTap: () => onSelectExecution?.call(exec.id),
                outputs: run.outputs
                    .where((o) => o.executionId == exec.id)
                    .toList(),
                scopeRuns: run.scopeRuns
                    .where((s) => s.parentExecutionId == exec.id)
                    .toList(),
                signalWaits: run.signalWaits
                    .where((w) => w.executionId == exec.id)
                    .toList(),
                signalMessages: run.signalMessages.toList(),
                onSelectInstance: onSelectInstance,
              )),
      ],
    );
  }
}

class _InstanceHeader extends StatelessWidget {
  const _InstanceHeader({required this.instance, required this.run});
  final rt.WorkflowInstance instance;
  final rt.GetInstanceRunResponse run;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final fmt = DateFormat.yMMMd().add_jm();

    return Container(
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
        color: theme.colorScheme.surfaceContainerLow,
        borderRadius: BorderRadius.circular(12),
        border: Border.all(color: theme.colorScheme.outlineVariant),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              TrustageStatusBadge(status: instance.status.name),
              const SizedBox(width: 8),
              Expanded(
                child: Text(
                  instance.workflowName,
                  style: theme.textTheme.titleSmall?.copyWith(
                    fontWeight: FontWeight.w600,
                  ),
                  overflow: TextOverflow.ellipsis,
                ),
              ),
            ],
          ),
          const SizedBox(height: 8),
          Wrap(
            spacing: 16,
            runSpacing: 4,
            children: [
              _MetaChip(label: 'ID', value: shortId(instance.id)),
              _MetaChip(label: 'State', value: instance.currentState),
              if (run.traceId.isNotEmpty)
                _MetaChip(label: 'Trace', value: shortId(run.traceId)),
              if (run.resumeStrategy.isNotEmpty)
                _MetaChip(
                  label: 'Resume',
                  value: humanizeStatus(run.resumeStrategy),
                ),
              _MetaChip(label: 'v${instance.workflowVersion}', value: ''),
              if (instance.hasStartedAt())
                _MetaChip(
                  label: 'Started',
                  value: fmt.format(instance.startedAt.toDateTime()),
                ),
            ],
          ),
          if (instance.parentInstanceId.isNotEmpty) ...[
            const SizedBox(height: 6),
            Row(
              children: [
                Icon(Icons.subdirectory_arrow_right,
                    size: 14, color: theme.colorScheme.onSurfaceVariant),
                const SizedBox(width: 4),
                Text(
                  'Child of ${shortId(instance.parentInstanceId)}',
                  style: theme.textTheme.labelSmall?.copyWith(
                    color: theme.colorScheme.primary,
                  ),
                ),
              ],
            ),
          ],
        ],
      ),
    );
  }
}

class _MetaChip extends StatelessWidget {
  const _MetaChip({required this.label, required this.value});
  final String label;
  final String value;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Row(
      mainAxisSize: MainAxisSize.min,
      children: [
        Text(
          label,
          style: theme.textTheme.labelSmall?.copyWith(
            color: theme.colorScheme.onSurfaceVariant,
            fontWeight: FontWeight.w500,
          ),
        ),
        if (value.isNotEmpty) ...[
          const SizedBox(width: 4),
          Text(
            value,
            style: theme.textTheme.labelSmall?.copyWith(
              fontFamily: 'monospace',
              fontWeight: FontWeight.w600,
            ),
          ),
        ],
      ],
    );
  }
}

class _ExecutionNode extends StatelessWidget {
  const _ExecutionNode({
    required this.execution,
    required this.isSelected,
    required this.onTap,
    required this.outputs,
    required this.scopeRuns,
    required this.signalWaits,
    required this.signalMessages,
    this.onSelectInstance,
  });

  final rt.WorkflowExecution execution;
  final bool isSelected;
  final VoidCallback onTap;
  final List<rt.StateOutput> outputs;
  final List<rt.ScopeRun> scopeRuns;
  final List<rt.SignalWait> signalWaits;
  final List<rt.SignalMessage> signalMessages;
  final ValueChanged<String>? onSelectInstance;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final status = execution.status.name;
    final color = statusColor(status);
    final fmt = DateFormat.Hm();

    return InkWell(
      onTap: onTap,
      borderRadius: BorderRadius.circular(10),
      child: Container(
        margin: const EdgeInsets.only(bottom: 4),
        padding: const EdgeInsets.all(12),
        decoration: BoxDecoration(
          color: isSelected
              ? color.withAlpha(15)
              : theme.colorScheme.surface,
          borderRadius: BorderRadius.circular(10),
          border: Border.all(
            color: isSelected ? color.withAlpha(60) : Colors.transparent,
          ),
        ),
        child: Row(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            // Timeline dot
            Padding(
              padding: const EdgeInsets.only(top: 2, right: 12),
              child: Container(
                width: 10,
                height: 10,
                decoration: BoxDecoration(
                  shape: BoxShape.circle,
                  color: color,
                ),
              ),
            ),

            // Content
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  // Header row
                  Row(
                    children: [
                      TrustageStatusBadge(status: status),
                      const SizedBox(width: 8),
                      Text(
                        execution.state,
                        style: theme.textTheme.titleSmall?.copyWith(
                          fontWeight: FontWeight.w600,
                        ),
                      ),
                      const SizedBox(width: 6),
                      Text(
                        '#${execution.attempt}',
                        style: theme.textTheme.labelSmall?.copyWith(
                          color: theme.colorScheme.onSurfaceVariant,
                        ),
                      ),
                    ],
                  ),
                  const SizedBox(height: 4),

                  // Metadata row
                  Wrap(
                    spacing: 12,
                    children: [
                      _MetaChip(label: 'ID', value: shortId(execution.id)),
                      if (execution.traceId.isNotEmpty)
                        _MetaChip(
                            label: 'Trace', value: shortId(execution.traceId)),
                      if (execution.hasStartedAt())
                        _MetaChip(
                          label: 'At',
                          value: fmt.format(execution.startedAt.toDateTime()),
                        ),
                    ],
                  ),

                  // Error message
                  if (execution.errorMessage.isNotEmpty) ...[
                    const SizedBox(height: 8),
                    Container(
                      width: double.infinity,
                      padding: const EdgeInsets.all(10),
                      decoration: BoxDecoration(
                        color: const Color(0xFFFFF1F2), // rose-50
                        borderRadius: BorderRadius.circular(8),
                        border: Border.all(
                            color: const Color(0xFFFECDD3)), // rose-200
                      ),
                      child: Text(
                        execution.errorMessage,
                        style: const TextStyle(
                            fontSize: 12, color: Color(0xFFBE123C)),
                      ),
                    ),
                  ],

                  // Nested details: outputs, scope runs, signals
                  ...outputs.map((o) => _OutputBox(output: o)),
                  ...scopeRuns.map((s) => _ScopeRunBox(scopeRun: s)),
                  ...signalWaits.map((w) => _SignalWaitBox(wait: w)),
                  ..._matchedMessages().map((m) => _SignalMessageBox(msg: m)),
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }

  List<rt.SignalMessage> _matchedMessages() {
    final waitIds = signalWaits.map((w) => w.id).toSet();
    return signalMessages
        .where((m) => waitIds.contains(m.waitId) || m.waitId.isEmpty)
        .toList();
  }
}

class _OutputBox extends StatelessWidget {
  const _OutputBox({required this.output});
  final rt.StateOutput output;

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.only(top: 8),
      child: Container(
        width: double.infinity,
        padding: const EdgeInsets.all(10),
        decoration: BoxDecoration(
          color: const Color(0xFFECFDF5), // emerald-50
          borderRadius: BorderRadius.circular(8),
          border: Border.all(color: const Color(0xFFA7F3D0)), // emerald-200
        ),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                const Icon(Icons.output, size: 14, color: Color(0xFF059669)),
                const SizedBox(width: 6),
                Text(
                  'Output: ${output.state}',
                  style: const TextStyle(
                    fontSize: 11,
                    fontWeight: FontWeight.w600,
                    color: Color(0xFF059669),
                  ),
                ),
              ],
            ),
            if (output.hasPayload()) ...[
              const SizedBox(height: 6),
              JsonBlock(value: _structToMap(output.payload), compact: true),
            ],
          ],
        ),
      ),
    );
  }
}

class _ScopeRunBox extends StatelessWidget {
  const _ScopeRunBox({required this.scopeRun});
  final rt.ScopeRun scopeRun;

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.only(top: 8),
      child: Container(
        width: double.infinity,
        padding: const EdgeInsets.all(10),
        decoration: BoxDecoration(
          color: const Color(0xFFF0F9FF), // sky-50
          borderRadius: BorderRadius.circular(8),
          border: Border.all(color: const Color(0xFFBAE6FD)), // sky-200
        ),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                const Icon(Icons.account_tree, size: 14, color: Color(0xFF0284C7)),
                const SizedBox(width: 6),
                TrustageStatusBadge(status: scopeRun.status),
                const SizedBox(width: 6),
                Text(
                  '${scopeRun.scopeType} scope',
                  style: const TextStyle(
                    fontSize: 11,
                    fontWeight: FontWeight.w600,
                    color: Color(0xFF0284C7),
                  ),
                ),
              ],
            ),
            const SizedBox(height: 4),
            Wrap(
              spacing: 12,
              children: [
                Text(
                  'Children: ${scopeRun.completedChildren}/${scopeRun.totalChildren}',
                  style: const TextStyle(fontSize: 11, color: Color(0xFF0369A1)),
                ),
                if (scopeRun.failedChildren > 0)
                  Text(
                    'Failed: ${scopeRun.failedChildren}',
                    style: const TextStyle(fontSize: 11, color: Color(0xFFE11D48)),
                  ),
                if (scopeRun.maxConcurrency > 0)
                  Text(
                    'Concurrency: ${scopeRun.maxConcurrency}',
                    style: const TextStyle(fontSize: 11, color: Color(0xFF0369A1)),
                  ),
              ],
            ),
          ],
        ),
      ),
    );
  }
}

class _SignalWaitBox extends StatelessWidget {
  const _SignalWaitBox({required this.wait});
  final rt.SignalWait wait;

  @override
  Widget build(BuildContext context) {
    final fmt = DateFormat.yMMMd().add_jm();

    return Padding(
      padding: const EdgeInsets.only(top: 8),
      child: Container(
        width: double.infinity,
        padding: const EdgeInsets.all(10),
        decoration: BoxDecoration(
          color: const Color(0xFFFFFBEB), // amber-50
          borderRadius: BorderRadius.circular(8),
          border: Border.all(color: const Color(0xFFFDE68A)), // amber-200
        ),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                const Icon(Icons.notifications_active,
                    size: 14, color: Color(0xFFD97706)),
                const SizedBox(width: 6),
                TrustageStatusBadge(status: wait.status),
                const SizedBox(width: 6),
                Text(
                  'Signal: ${wait.signalName}',
                  style: const TextStyle(
                    fontSize: 11,
                    fontWeight: FontWeight.w600,
                    color: Color(0xFFD97706),
                  ),
                ),
              ],
            ),
            const SizedBox(height: 4),
            Wrap(
              spacing: 12,
              children: [
                if (wait.outputVar.isNotEmpty)
                  Text(
                    'Output → ${wait.outputVar}',
                    style: const TextStyle(fontSize: 11, color: Color(0xFF92400E)),
                  ),
                if (wait.hasTimeoutAt())
                  Text(
                    'Timeout: ${fmt.format(wait.timeoutAt.toDateTime())}',
                    style: const TextStyle(fontSize: 11, color: Color(0xFF92400E)),
                  ),
                Text(
                  'Attempts: ${wait.attempts}',
                  style: const TextStyle(fontSize: 11, color: Color(0xFF92400E)),
                ),
              ],
            ),
          ],
        ),
      ),
    );
  }
}

class _SignalMessageBox extends StatelessWidget {
  const _SignalMessageBox({required this.msg});
  final rt.SignalMessage msg;

  @override
  Widget build(BuildContext context) {
    final fmt = DateFormat.Hm();

    return Padding(
      padding: const EdgeInsets.only(top: 8),
      child: Container(
        width: double.infinity,
        padding: const EdgeInsets.all(10),
        decoration: BoxDecoration(
          color: const Color(0xFFF5F3FF), // violet-50
          borderRadius: BorderRadius.circular(8),
          border: Border.all(color: const Color(0xFFDDD6FE)), // violet-200
        ),
        child: Row(
          children: [
            const Icon(Icons.send, size: 14, color: Color(0xFF7C3AED)),
            const SizedBox(width: 6),
            TrustageStatusBadge(status: msg.status),
            const SizedBox(width: 6),
            Text(
              msg.signalName,
              style: const TextStyle(
                fontSize: 11,
                fontWeight: FontWeight.w600,
                color: Color(0xFF7C3AED),
              ),
            ),
            const Spacer(),
            if (msg.hasDeliveredAt())
              Text(
                fmt.format(msg.deliveredAt.toDateTime()),
                style: const TextStyle(fontSize: 11, color: Color(0xFF6D28D9)),
              ),
          ],
        ),
      ),
    );
  }
}
