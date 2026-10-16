# JNI resolves generated binding classes and native entry points by name.
-keep class go.** { *; }
-keep class com.olivierh.passingshot.mobile.** { *; }
-keep class com.olivierh.passingshot.ebitenmobileview.** { *; }
-keepclasseswithmembernames,includedescriptorclasses class * {
    native <methods>;
}
